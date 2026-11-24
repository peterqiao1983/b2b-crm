package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"b2b-crm/internal/db"
	"b2b-crm/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func toolSchemas() []toolDef {
	return []toolDef{
		{
			Name:        "crm_dashboard",
			Description: "获取 CRM 工作台总览统计数据（客户数、项目数、商机金额、近期跟进待办计划）",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "crm_search_companies",
			Description: "搜索或列出客户公司，支持按关键词和阶段过滤",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"q":     map[string]interface{}{"type": "string", "description": "搜索关键词（公司名/行业/负责人）"},
					"stage": map[string]interface{}{"type": "string", "description": "阶段过滤: lead / customer / key_account"},
				},
			},
		},
		{
			Name:        "crm_get_company_summary",
			Description: "获取指定公司的完整链式汇总视图：公司信息、旗下所有项目及专属人员角色态度、联系人跨项目轨迹、最新20条跟进动态",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"company_id": map[string]interface{}{"type": "integer", "description": "公司 ID"},
				},
				"required": []string{"company_id"},
			},
		},
		{
			Name:        "crm_create_company",
			Description: "新建客户公司",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":     map[string]interface{}{"type": "string", "description": "公司名称"},
					"stage":    map[string]interface{}{"type": "string", "description": "阶段: lead(默认) / customer / key_account"},
					"industry": map[string]interface{}{"type": "string", "description": "行业"},
					"owner":    map[string]interface{}{"type": "string", "description": "负责人"},
					"source":   map[string]interface{}{"type": "string", "description": "线索来源"},
					"notes":    map[string]interface{}{"type": "string", "description": "备注"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "crm_create_contact",
			Description: "新建联系人（所属公司可选，若暂无对应公司可留空后期通过 crm_update_contact 补全）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"company_id": map[string]interface{}{"type": "integer", "description": "所属公司 ID（可选，不传则暂不关联公司）"},
					"name":       map[string]interface{}{"type": "string", "description": "联系人姓名"},
					"title":      map[string]interface{}{"type": "string", "description": "职位"},
					"phone":      map[string]interface{}{"type": "string", "description": "电话"},
					"wechat":     map[string]interface{}{"type": "string", "description": "微信"},
					"email":      map[string]interface{}{"type": "string", "description": "邮箱"},
					"notes":      map[string]interface{}{"type": "string", "description": "备注"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "crm_list_contacts",
			Description: "查询联系人列表，支持关键词搜索或仅查看未关联公司的独立联系人",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"q":             map[string]interface{}{"type": "string", "description": "关键词（姓名/电话/职位）"},
					"unlinked_only": map[string]interface{}{"type": "boolean", "description": "是否仅看未关联公司的联系人"},
				},
			},
		},
		{
			Name:        "crm_create_project",
			Description: "为指定公司新建项目/商机",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"company_id":       map[string]interface{}{"type": "integer", "description": "所属公司 ID"},
					"name":             map[string]interface{}{"type": "string", "description": "项目名称"},
					"stage":            map[string]interface{}{"type": "string", "description": "阶段: initial(默认) / testing / quoting / won / lost"},
					"estimated_amount": map[string]interface{}{"type": "number", "description": "预估金额"},
					"description":      map[string]interface{}{"type": "string", "description": "项目说明"},
				},
				"required": []string{"company_id", "name"},
			},
		},
		{
			Name:        "crm_assign_project_contact",
			Description: "将公司联系人指派到指定项目，并设置其在该项目中的角色与态度",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]interface{}{"type": "integer", "description": "项目 ID"},
					"contact_id": map[string]interface{}{"type": "integer", "description": "联系人 ID"},
					"role":       map[string]interface{}{"type": "string", "description": "项目角色（如：决策人/技术评估/商务接口）"},
					"attitude":   map[string]interface{}{"type": "string", "description": "态度: support / neutral(默认) / oppose"},
					"notes":      map[string]interface{}{"type": "string", "description": "备注"},
				},
				"required": []string{"project_id", "contact_id"},
			},
		},
		{
			Name:        "crm_add_follow_up",
			Description: "记录一条跟进纪要，并可选设置下次跟进计划时间与待办备注",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"company_id":       map[string]interface{}{"type": "integer", "description": "所属公司 ID"},
					"project_id":       map[string]interface{}{"type": "integer", "description": "关联项目 ID（可选）"},
					"contact_id":       map[string]interface{}{"type": "integer", "description": "对接人 ID（可选）"},
					"follow_type":      map[string]interface{}{"type": "string", "description": "方式: visit(默认) / call / meeting / sample / other"},
					"summary":          map[string]interface{}{"type": "string", "description": "跟进纪要"},
					"occurred_at":      map[string]interface{}{"type": "string", "description": "发生时间（RFC3339 或 YYYY-MM-DD HH:MM，默认当前时间）"},
					"next_follow_at":   map[string]interface{}{"type": "string", "description": "下次跟进时间（RFC3339 或 YYYY-MM-DD HH:MM，可选）"},
					"next_follow_note": map[string]interface{}{"type": "string", "description": "下次跟进待办备注（可选）"},
				},
				"required": []string{"company_id", "summary"},
			},
		},
		{
			Name:        "crm_calendar_follow_ups",
			Description: "按日期区间查询已完成跟进与未来待办跟进计划",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"start": map[string]interface{}{"type": "string", "description": "开始日期 (YYYY-MM-DD)"},
					"end":   map[string]interface{}{"type": "string", "description": "结束日期 (YYYY-MM-DD)"},
				},
			},
		},
		{
			Name:        "crm_update_company",
			Description: "更新客户公司信息（仅更新传入的字段）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"company_id": map[string]interface{}{"type": "integer", "description": "公司 ID"},
					"name":       map[string]interface{}{"type": "string", "description": "公司名称"},
					"stage":      map[string]interface{}{"type": "string", "description": "阶段: lead / customer / key_account"},
					"industry":   map[string]interface{}{"type": "string", "description": "行业"},
					"owner":      map[string]interface{}{"type": "string", "description": "负责人"},
					"source":     map[string]interface{}{"type": "string", "description": "线索来源"},
					"notes":      map[string]interface{}{"type": "string", "description": "备注"},
				},
				"required": []string{"company_id"},
			},
		},
		{
			Name:        "crm_update_contact",
			Description: "更新联系人信息（仅更新传入的字段）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"contact_id": map[string]interface{}{"type": "integer", "description": "联系人 ID"},
					"company_id": map[string]interface{}{"type": "integer", "description": "所属公司 ID"},
					"name":       map[string]interface{}{"type": "string", "description": "姓名"},
					"title":      map[string]interface{}{"type": "string", "description": "职位"},
					"phone":      map[string]interface{}{"type": "string", "description": "电话"},
					"wechat":     map[string]interface{}{"type": "string", "description": "微信"},
					"email":      map[string]interface{}{"type": "string", "description": "邮箱"},
					"notes":      map[string]interface{}{"type": "string", "description": "备注"},
				},
				"required": []string{"contact_id"},
			},
		},
		{
			Name:        "crm_update_project",
			Description: "更新项目/商机信息（仅更新传入的字段）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id":       map[string]interface{}{"type": "integer", "description": "项目 ID"},
					"name":             map[string]interface{}{"type": "string", "description": "项目名称"},
					"stage":            map[string]interface{}{"type": "string", "description": "阶段: initial / testing / quoting / won / lost"},
					"estimated_amount": map[string]interface{}{"type": "number", "description": "预估金额"},
					"description":      map[string]interface{}{"type": "string", "description": "项目说明"},
				},
				"required": []string{"project_id"},
			},
		},
		{
			Name:        "crm_update_follow_up",
			Description: "更新跟进记录（仅更新传入的字段）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"follow_up_id":     map[string]interface{}{"type": "integer", "description": "跟进记录 ID"},
					"project_id":       map[string]interface{}{"type": "integer", "description": "关联项目 ID"},
					"contact_id":       map[string]interface{}{"type": "integer", "description": "对接人 ID"},
					"follow_type":      map[string]interface{}{"type": "string", "description": "方式: visit / call / meeting / sample / other"},
					"summary":          map[string]interface{}{"type": "string", "description": "跟进纪要"},
					"occurred_at":      map[string]interface{}{"type": "string", "description": "发生时间（RFC3339 或 YYYY-MM-DD HH:MM）"},
					"next_follow_at":   map[string]interface{}{"type": "string", "description": "下次跟进时间（RFC3339 或 YYYY-MM-DD HH:MM）"},
					"next_follow_note": map[string]interface{}{"type": "string", "description": "下次跟进待办备注"},
				},
				"required": []string{"follow_up_id"},
			},
		},
	}
}

func parseTimeFlex(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.Local
	}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间格式: %s", s)
}

func callTool(database *gorm.DB, name string, rawArgs json.RawMessage) (string, bool) {
	var args map[string]interface{}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args == nil {
		args = map[string]interface{}{}
	}
	strArg := func(k string) string {
		if v, ok := args[k].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	uintArg := func(k string) uint {
		if v, ok := args[k].(float64); ok && v > 0 {
			return uint(v)
		}
		return 0
	}
	floatArg := func(k string) float64 {
		if v, ok := args[k].(float64); ok {
			return v
		}
		return 0
	}

	var result interface{}
	switch name {
	case "crm_dashboard":
		var companyCount, projectCount, contactCount, followUpCount int64
		database.Model(&models.Company{}).Count(&companyCount)
		database.Model(&models.Project{}).Count(&projectCount)
		database.Model(&models.Contact{}).Count(&contactCount)
		database.Model(&models.FollowUp{}).Count(&followUpCount)
		var totalAmount, wonAmount float64
		database.Model(&models.Project{}).Where("stage != ?", "lost").Select("COALESCE(SUM(estimated_amount), 0)").Scan(&totalAmount)
		database.Model(&models.Project{}).Where("stage = ?", "won").Select("COALESCE(SUM(estimated_amount), 0)").Scan(&wonAmount)

		type upcomingItem struct {
			models.FollowUp
			CompanyName string `json:"company_name"`
			ProjectName string `json:"project_name"`
			ContactName string `json:"contact_name"`
		}
		var upcoming []upcomingItem
		database.Table("follow_ups").
			Select("follow_ups.*, companies.name as company_name, COALESCE(projects.name, '') as project_name, COALESCE(contacts.name, '') as contact_name").
			Joins("LEFT JOIN companies ON companies.id = follow_ups.company_id").
			Joins("LEFT JOIN projects ON projects.id = follow_ups.project_id").
			Joins("LEFT JOIN contacts ON contacts.id = follow_ups.contact_id").
			Where("follow_ups.next_follow_at IS NOT NULL AND follow_ups.next_follow_at >= ?", time.Now().Add(-24*time.Hour)).
			Order("follow_ups.next_follow_at ASC").
			Limit(20).
			Scan(&upcoming)
		result = map[string]interface{}{
			"counts": map[string]interface{}{
				"companies": companyCount, "projects": projectCount, "contacts": contactCount,
				"follow_ups": followUpCount, "estimated_amount": totalAmount, "won_amount": wonAmount,
			},
			"upcoming": upcoming,
		}

	case "crm_search_companies":
		q := strArg("q")
		stage := strArg("stage")
		tx := database.Model(&models.Company{})
		if q != "" {
			like := "%" + q + "%"
			tx = tx.Where("name LIKE ? OR industry LIKE ? OR owner LIKE ?", like, like, like)
		}
		if stage != "" {
			tx = tx.Where("stage = ?", stage)
		}
		var list []models.Company
		if err := tx.Order("updated_at DESC").Limit(50).Find(&list).Error; err != nil {
			return err.Error(), true
		}
		result = list

	case "crm_get_company_summary":
		cid := uintArg("company_id")
		if cid == 0 {
			return "company_id 必填", true
		}
		var company models.Company
		if err := database.First(&company, cid).Error; err != nil {
			return "公司不存在", true
		}
		var projects []models.Project
		var contacts []models.Contact
		var followUps []models.FollowUp
		database.Where("company_id = ?", cid).Order("updated_at DESC").Find(&projects)
		database.Where("company_id = ?", cid).Order("id ASC").Find(&contacts)
		database.Where("company_id = ?", cid).Order("occurred_at DESC").Limit(30).Find(&followUps)

		var projIDs []uint
		for _, p := range projects {
			projIDs = append(projIDs, p.ID)
		}
		var links []models.ProjectContact
		if len(projIDs) > 0 {
			database.Where("project_id IN ?", projIDs).Find(&links)
		}
		result = map[string]interface{}{
			"company":          company,
			"projects":         projects,
			"contacts":         contacts,
			"project_contacts": links,
			"recent_follow_ups": followUps,
		}

	case "crm_create_company":
		cName := strArg("name")
		if cName == "" {
			return "name 必填", true
		}
		stage := strArg("stage")
		if stage == "" {
			stage = "lead"
		}
		comp := models.Company{
			Name:     cName,
			Stage:    stage,
			Industry: strArg("industry"),
			Owner:    strArg("owner"),
			Source:   strArg("source"),
			Notes:    strArg("notes"),
		}
		if err := database.Create(&comp).Error; err != nil {
			return err.Error(), true
		}
		result = comp

	case "crm_create_contact":
		cid := uintArg("company_id")
		cName := strArg("name")
		if cName == "" {
			return "name 必填", true
		}
		var cidPtr *uint
		if cid > 0 {
			cidPtr = &cid
		}
		ct := models.Contact{
			CompanyID: cidPtr,
			Name:      cName,
			Title:     strArg("title"),
			Phone:     strArg("phone"),
			Wechat:    strArg("wechat"),
			Email:     strArg("email"),
			Notes:     strArg("notes"),
		}
		if err := database.Create(&ct).Error; err != nil {
			return err.Error(), true
		}
		result = ct

	case "crm_list_contacts":
		q := strArg("q")
		unlinkedOnly, _ := args["unlinked_only"].(bool)
		type contactRow struct {
			models.Contact
			CompanyName string `json:"company_name"`
		}
		tx := database.Table("contacts").
			Select("contacts.*, COALESCE(companies.name, '') as company_name").
			Joins("LEFT JOIN companies ON companies.id = contacts.company_id")
		if unlinkedOnly {
			tx = tx.Where("contacts.company_id IS NULL OR contacts.company_id = 0")
		}
		if q != "" {
			like := "%" + q + "%"
			tx = tx.Where("contacts.name LIKE ? OR contacts.phone LIKE ? OR contacts.title LIKE ? OR companies.name LIKE ?", like, like, like, like)
		}
		var list []contactRow
		if err := tx.Order("contacts.updated_at DESC").Limit(50).Scan(&list).Error; err != nil {
			return err.Error(), true
		}
		result = list

	case "crm_create_project":
		cid := uintArg("company_id")
		pName := strArg("name")
		if cid == 0 || pName == "" {
			return "company_id 和 name 必填", true
		}
		stage := strArg("stage")
		if stage == "" {
			stage = "initial"
		}
		proj := models.Project{
			CompanyID:       cid,
			Name:            pName,
			Stage:           stage,
			EstimatedAmount: floatArg("estimated_amount"),
			Description:     strArg("description"),
		}
		if err := database.Create(&proj).Error; err != nil {
			return err.Error(), true
		}
		result = proj

	case "crm_assign_project_contact":
		pid := uintArg("project_id")
		ctid := uintArg("contact_id")
		if pid == 0 || ctid == 0 {
			return "project_id 和 contact_id 必填", true
		}
		att := strArg("attitude")
		if att == "" {
			att = "neutral"
		}
		link := models.ProjectContact{
			ProjectID: pid,
			ContactID: ctid,
			Role:      strArg("role"),
			Attitude:  att,
			Notes:     strArg("notes"),
		}
		if err := database.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "project_id"}, {Name: "contact_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"role", "attitude", "notes"}),
		}).Create(&link).Error; err != nil {
			return err.Error(), true
		}
		result = link

	case "crm_add_follow_up":
		cid := uintArg("company_id")
		summary := strArg("summary")
		if cid == 0 || summary == "" {
			return "company_id 和 summary 必填", true
		}
		ft := strArg("follow_type")
		if ft == "" {
			ft = "visit"
		}
		occ := time.Now()
		if s := strArg("occurred_at"); s != "" {
			t, err := parseTimeFlex(s)
			if err != nil {
				return err.Error(), true
			}
			occ = t
		}
		var nextAt *time.Time
		if s := strArg("next_follow_at"); s != "" {
			t, err := parseTimeFlex(s)
			if err != nil {
				return err.Error(), true
			}
			nextAt = &t
		}
		var pidPtr, ctidPtr *uint
		if pid := uintArg("project_id"); pid > 0 {
			pidPtr = &pid
		}
		if ctid := uintArg("contact_id"); ctid > 0 {
			ctidPtr = &ctid
		}
		fu := models.FollowUp{
			CompanyID:      cid,
			ProjectID:      pidPtr,
			ContactID:      ctidPtr,
			FollowType:     ft,
			OccurredAt:     occ,
			NextFollowAt:   nextAt,
			NextFollowNote: strArg("next_follow_note"),
			Summary:        summary,
			CreatedBy:      "work-assistant",
		}
		if err := database.Create(&fu).Error; err != nil {
			return err.Error(), true
		}
		result = fu

	case "crm_calendar_follow_ups":
		now := time.Now()
		start := now.AddDate(0, 0, -7)
		end := now.AddDate(0, 0, 14)
		if s := strArg("start"); s != "" {
			if t, err := parseTimeFlex(s); err == nil {
				start = t
			}
		}
		if s := strArg("end"); s != "" {
			if t, err := parseTimeFlex(s); err == nil {
				end = t.Add(24 * time.Hour)
			}
		}
		type fuJoin struct {
			models.FollowUp
			CompanyName string `json:"company_name"`
			ProjectName string `json:"project_name"`
			ContactName string `json:"contact_name"`
		}
		var completed, upcoming []fuJoin
		database.Table("follow_ups").
			Select("follow_ups.*, COALESCE(companies.name, '') as company_name, COALESCE(projects.name, '') as project_name, COALESCE(contacts.name, '') as contact_name").
			Joins("LEFT JOIN companies ON companies.id = follow_ups.company_id").
			Joins("LEFT JOIN projects ON projects.id = follow_ups.project_id").
			Joins("LEFT JOIN contacts ON contacts.id = follow_ups.contact_id").
			Where("follow_ups.occurred_at >= ? AND follow_ups.occurred_at <= ?", start, end).
			Order("follow_ups.occurred_at ASC").Scan(&completed)
		database.Table("follow_ups").
			Select("follow_ups.*, COALESCE(companies.name, '') as company_name, COALESCE(projects.name, '') as project_name, COALESCE(contacts.name, '') as contact_name").
			Joins("LEFT JOIN companies ON companies.id = follow_ups.company_id").
			Joins("LEFT JOIN projects ON projects.id = follow_ups.project_id").
			Joins("LEFT JOIN contacts ON contacts.id = follow_ups.contact_id").
			Where("follow_ups.next_follow_at IS NOT NULL AND follow_ups.next_follow_at >= ? AND follow_ups.next_follow_at <= ?", start, end).
			Order("follow_ups.next_follow_at ASC").Scan(&upcoming)
		result = map[string]interface{}{"completed": completed, "upcoming": upcoming}

	case "crm_update_company":
		cid := uintArg("company_id")
		if cid == 0 {
			return "company_id 必填", true
		}
		var comp models.Company
		if err := database.First(&comp, cid).Error; err != nil {
			return "公司不存在", true
		}
		updates := map[string]interface{}{}
		for _, k := range []string{"name", "stage", "industry", "owner", "source", "notes"} {
			if _, ok := args[k]; ok {
				updates[k] = strArg(k)
			}
		}
		if len(updates) > 0 {
			if err := database.Model(&comp).Updates(updates).Error; err != nil {
				return err.Error(), true
			}
			_ = database.First(&comp, cid)
		}
		result = comp

	case "crm_update_contact":
		ctid := uintArg("contact_id")
		if ctid == 0 {
			return "contact_id 必填", true
		}
		var ct models.Contact
		if err := database.First(&ct, ctid).Error; err != nil {
			return "联系人不存在", true
		}
		updates := map[string]interface{}{}
		for _, k := range []string{"name", "title", "phone", "wechat", "email", "notes"} {
			if _, ok := args[k]; ok {
				updates[k] = strArg(k)
			}
		}
		if cid := uintArg("company_id"); cid > 0 {
			updates["company_id"] = cid
		}
		if len(updates) > 0 {
			if err := database.Model(&ct).Updates(updates).Error; err != nil {
				return err.Error(), true
			}
			_ = database.First(&ct, ctid)
		}
		result = ct

	case "crm_update_project":
		pid := uintArg("project_id")
		if pid == 0 {
			return "project_id 必填", true
		}
		var proj models.Project
		if err := database.First(&proj, pid).Error; err != nil {
			return "项目不存在", true
		}
		updates := map[string]interface{}{}
		for _, k := range []string{"name", "stage", "description"} {
			if _, ok := args[k]; ok {
				updates[k] = strArg(k)
			}
		}
		if _, ok := args["estimated_amount"]; ok {
			updates["estimated_amount"] = floatArg("estimated_amount")
		}
		if len(updates) > 0 {
			if err := database.Model(&proj).Updates(updates).Error; err != nil {
				return err.Error(), true
			}
			_ = database.First(&proj, pid)
		}
		result = proj

	case "crm_update_follow_up":
		fid := uintArg("follow_up_id")
		if fid == 0 {
			return "follow_up_id 必填", true
		}
		var fu models.FollowUp
		if err := database.First(&fu, fid).Error; err != nil {
			return "跟进记录不存在", true
		}
		updates := map[string]interface{}{}
		for _, k := range []string{"follow_type", "summary", "next_follow_note"} {
			if _, ok := args[k]; ok {
				updates[k] = strArg(k)
			}
		}
		if pid := uintArg("project_id"); pid > 0 {
			updates["project_id"] = pid
		}
		if ctid := uintArg("contact_id"); ctid > 0 {
			updates["contact_id"] = ctid
		}
		if s := strArg("occurred_at"); s != "" {
			t, err := parseTimeFlex(s)
			if err != nil {
				return err.Error(), true
			}
			updates["occurred_at"] = t
		}
		if s := strArg("next_follow_at"); s != "" {
			t, err := parseTimeFlex(s)
			if err != nil {
				return err.Error(), true
			}
			updates["next_follow_at"] = t
		}
		if len(updates) > 0 {
			if err := database.Model(&fu).Updates(updates).Error; err != nil {
				return err.Error(), true
			}
			_ = database.First(&fu, fid)
		}
		result = fu

	default:
		return "未知工具: " + name, true
	}

	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err.Error(), true
	}
	return string(b), false
}

func main() {
	log.SetOutput(os.Stderr)
	dbPath := getEnv("CRM_DB_PATH", "/opt/crm/data/crm.db")
	database, err := db.Init(dbPath)
	if err != nil {
		log.Fatalf("init db failed: %v", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		// Notification without id -> no response
		if len(req.ID) == 0 || string(req.ID) == "null" {
			continue
		}

		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "initialize":
			resp.Result = map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "b2b-crm-mcp",
					"version": "1.0.0",
				},
			}
		case "ping":
			resp.Result = map[string]interface{}{}
		case "tools/list":
			resp.Result = map[string]interface{}{
				"tools": toolSchemas(),
			}
		case "tools/call":
			var p struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				resp.Error = &rpcError{Code: -32602, Message: "invalid params"}
			} else {
				text, isErr := callTool(database, p.Name, p.Arguments)
				resp.Result = map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": text},
					},
					"isError": isErr,
				}
			}
		default:
			resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
		}
		if err := enc.Encode(resp); err != nil {
			log.Printf("encode error: %v", err)
		}
	}
}
