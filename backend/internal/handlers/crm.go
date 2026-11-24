package handlers

import (
	"net/http"
	"strconv"
	"time"

	"b2b-crm/internal/models"
	"b2b-crm/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CRMHandler struct {
	DB *gorm.DB
}

// ---------- Dashboard ----------

func (h *CRMHandler) Dashboard(c *gin.Context) {
	var companyCount, projectCount, contactCount, followUpCount int64
	h.DB.Model(&models.Company{}).Count(&companyCount)
	h.DB.Model(&models.Project{}).Count(&projectCount)
	h.DB.Model(&models.Contact{}).Count(&contactCount)
	h.DB.Model(&models.FollowUp{}).Count(&followUpCount)

	var totalAmount float64
	h.DB.Model(&models.Project{}).Where("stage != ?", "lost").Select("COALESCE(SUM(estimated_amount), 0)").Scan(&totalAmount)

	var wonAmount float64
	h.DB.Model(&models.Project{}).Where("stage = ?", "won").Select("COALESCE(SUM(estimated_amount), 0)").Scan(&wonAmount)

	type upcomingItem struct {
		models.FollowUp
		CompanyName string `json:"company_name"`
		ProjectName string `json:"project_name"`
		ContactName string `json:"contact_name"`
	}
	var upcoming []upcomingItem
	now := time.Now().Add(-24 * time.Hour)
	h.DB.Table("follow_ups").
		Select("follow_ups.*, companies.name as company_name, COALESCE(projects.name, '') as project_name, COALESCE(contacts.name, '') as contact_name").
		Joins("LEFT JOIN companies ON companies.id = follow_ups.company_id").
		Joins("LEFT JOIN projects ON projects.id = follow_ups.project_id").
		Joins("LEFT JOIN contacts ON contacts.id = follow_ups.contact_id").
		Where("follow_ups.next_follow_at IS NOT NULL AND follow_ups.next_follow_at >= ?", now).
		Order("follow_ups.next_follow_at ASC").
		Limit(20).
		Scan(&upcoming)

	response.OK(c, gin.H{
		"counts": gin.H{
			"companies":        companyCount,
			"projects":         projectCount,
			"contacts":         contactCount,
			"follow_ups":       followUpCount,
			"estimated_amount": totalAmount,
			"won_amount":       wonAmount,
		},
		"upcoming": upcoming,
	})
}

// ---------- Companies ----------

func (h *CRMHandler) ListCompanies(c *gin.Context) {
	q := c.Query("q")
	stage := c.Query("stage")

	tx := h.DB.Model(&models.Company{})
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("name LIKE ? OR industry LIKE ? OR owner LIKE ?", like, like, like)
	}
	if stage != "" {
		tx = tx.Where("stage = ?", stage)
	}

	var list []models.Company
	if err := tx.Order("updated_at DESC").Find(&list).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "查询公司列表失败")
		return
	}

	// Attach project count & contact count without N+1
	type countRow struct {
		CompanyID uint
		Cnt       int64
	}
	projCounts := map[uint]int64{}
	var pRows []countRow
	h.DB.Model(&models.Project{}).Select("company_id, COUNT(*) as cnt").Group("company_id").Scan(&pRows)
	for _, r := range pRows {
		projCounts[r.CompanyID] = r.Cnt
	}

	contCounts := map[uint]int64{}
	var cRows []countRow
	h.DB.Model(&models.Contact{}).Select("company_id, COUNT(*) as cnt").Group("company_id").Scan(&cRows)
	for _, r := range cRows {
		contCounts[r.CompanyID] = r.Cnt
	}

	type companyItem struct {
		models.Company
		ProjectCount int64 `json:"project_count"`
		ContactCount int64 `json:"contact_count"`
	}
	res := make([]companyItem, 0, len(list))
	for _, comp := range list {
		res = append(res, companyItem{
			Company:      comp,
			ProjectCount: projCounts[comp.ID],
			ContactCount: contCounts[comp.ID],
		})
	}
	response.OK(c, res)
}

func (h *CRMHandler) CreateCompany(c *gin.Context) {
	var item models.Company
	if err := c.ShouldBindJSON(&item); err != nil || item.Name == "" {
		response.Error(c, http.StatusBadRequest, "公司名称不能为空")
		return
	}
	if item.Stage == "" {
		item.Stage = "lead"
	}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建公司失败")
		return
	}
	response.OK(c, item)
}

func (h *CRMHandler) UpdateCompany(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的公司 ID")
		return
	}
	var existing models.Company
	if err := h.DB.First(&existing, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "公司不存在")
		return
	}
	var payload models.Company
	if err := c.ShouldBindJSON(&payload); err != nil || payload.Name == "" {
		response.Error(c, http.StatusBadRequest, "公司名称不能为空")
		return
	}
	existing.Name = payload.Name
	if payload.Stage != "" {
		existing.Stage = payload.Stage
	}
	existing.Industry = payload.Industry
	existing.Owner = payload.Owner
	existing.Source = payload.Source
	existing.Website = payload.Website
	existing.Address = payload.Address
	existing.Notes = payload.Notes

	if err := h.DB.Save(&existing).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "更新公司失败")
		return
	}
	response.OK(c, existing)
}

func (h *CRMHandler) DeleteCompany(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的公司 ID")
		return
	}
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var projIDs []uint
		tx.Model(&models.Project{}).Where("company_id = ?", id).Pluck("id", &projIDs)
		if len(projIDs) > 0 {
			tx.Where("project_id IN ?", projIDs).Delete(&models.ProjectContact{})
		}
		tx.Where("company_id = ?", id).Delete(&models.FollowUp{})
		tx.Where("company_id = ?", id).Delete(&models.Project{})
		tx.Where("company_id = ?", id).Delete(&models.Contact{})
		return tx.Delete(&models.Company{}, id).Error
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "删除公司失败")
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// ChainSummary: 同一公司 -> 不同项目 -> 不同人员 链式汇总穿透接口
func (h *CRMHandler) CompanyChainSummary(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的公司 ID")
		return
	}

	var company models.Company
	if err := h.DB.First(&company, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "公司不存在")
		return
	}

	var projects []models.Project
	h.DB.Where("company_id = ?", id).Order("updated_at DESC").Find(&projects)

	var contacts []models.Contact
	h.DB.Where("company_id = ?", id).Order("id ASC").Find(&contacts)

	var followUps []models.FollowUp
	h.DB.Where("company_id = ?", id).Order("occurred_at DESC").Find(&followUps)

	projIDs := make([]uint, 0, len(projects))
	projMap := make(map[uint]models.Project, len(projects))
	for _, p := range projects {
		projIDs = append(projIDs, p.ID)
		projMap[p.ID] = p
	}
	contactMap := make(map[uint]models.Contact, len(contacts))
	for _, ct := range contacts {
		contactMap[ct.ID] = ct
	}

	var links []models.ProjectContact
	if len(projIDs) > 0 {
		h.DB.Where("project_id IN ?", projIDs).Find(&links)
	}

	type projectContactView struct {
		LinkID    uint   `json:"link_id"`
		ContactID uint   `json:"contact_id"`
		Name      string `json:"name"`
		Title     string `json:"title"`
		Phone     string `json:"phone"`
		Email     string `json:"email"`
		Wechat    string `json:"wechat"`
		Role      string `json:"role"`
		Attitude  string `json:"attitude"`
		Notes     string `json:"notes"`
	}
	type followUpView struct {
		models.FollowUp
		ProjectName string `json:"project_name"`
		ContactName string `json:"contact_name"`
	}
	type projectSummary struct {
		models.Project
		Contacts  []projectContactView `json:"contacts"`
		FollowUps []followUpView       `json:"follow_ups"`
	}
	type contactProjectRole struct {
		ProjectID   uint   `json:"project_id"`
		ProjectName string `json:"project_name"`
		Stage       string `json:"stage"`
		Role        string `json:"role"`
		Attitude    string `json:"attitude"`
	}
	type contactSummary struct {
		models.Contact
		Projects  []contactProjectRole `json:"projects"`
		FollowUps []followUpView       `json:"follow_ups"`
	}

	// Build follow-up views & group in memory
	allFUViews := make([]followUpView, 0, len(followUps))
	fuByProject := map[uint][]followUpView{}
	fuByContact := map[uint][]followUpView{}
	for _, fu := range followUps {
		v := followUpView{FollowUp: fu}
		if fu.ProjectID != nil {
			if p, ok := projMap[*fu.ProjectID]; ok {
				v.ProjectName = p.Name
			}
			fuByProject[*fu.ProjectID] = append(fuByProject[*fu.ProjectID], v)
		}
		if fu.ContactID != nil {
			if ct, ok := contactMap[*fu.ContactID]; ok {
				v.ContactName = ct.Name
			}
			fuByContact[*fu.ContactID] = append(fuByContact[*fu.ContactID], v)
		}
		allFUViews = append(allFUViews, v)
	}

	// Group links
	linksByProj := map[uint][]projectContactView{}
	rolesByContact := map[uint][]contactProjectRole{}
	for _, l := range links {
		if ct, ok := contactMap[l.ContactID]; ok {
			linksByProj[l.ProjectID] = append(linksByProj[l.ProjectID], projectContactView{
				LinkID:    l.ID,
				ContactID: ct.ID,
				Name:      ct.Name,
				Title:     ct.Title,
				Phone:     ct.Phone,
				Email:     ct.Email,
				Wechat:    ct.Wechat,
				Role:      l.Role,
				Attitude:  l.Attitude,
				Notes:     l.Notes,
			})
		}
		if p, ok := projMap[l.ProjectID]; ok {
			rolesByContact[l.ContactID] = append(rolesByContact[l.ContactID], contactProjectRole{
				ProjectID:   p.ID,
				ProjectName: p.Name,
				Stage:       p.Stage,
				Role:        l.Role,
				Attitude:    l.Attitude,
			})
		}
	}

	var totalAmount, wonAmount float64
	activeProjects := 0
	projSummaries := make([]projectSummary, 0, len(projects))
	for _, p := range projects {
		if p.Stage != "lost" {
			totalAmount += p.EstimatedAmount
		}
		if p.Stage == "won" {
			wonAmount += p.EstimatedAmount
		}
		if p.Stage != "won" && p.Stage != "lost" {
			activeProjects++
		}
		cList := linksByProj[p.ID]
		if cList == nil {
			cList = []projectContactView{}
		}
		fList := fuByProject[p.ID]
		if fList == nil {
			fList = []followUpView{}
		}
		projSummaries = append(projSummaries, projectSummary{
			Project:   p,
			Contacts:  cList,
			FollowUps: fList,
		})
	}

	contactSummaries := make([]contactSummary, 0, len(contacts))
	for _, ct := range contacts {
		pRoles := rolesByContact[ct.ID]
		if pRoles == nil {
			pRoles = []contactProjectRole{}
		}
		fList := fuByContact[ct.ID]
		if fList == nil {
			fList = []followUpView{}
		}
		contactSummaries = append(contactSummaries, contactSummary{
			Contact:   ct,
			Projects:  pRoles,
			FollowUps: fList,
		})
	}

	recent := allFUViews
	if len(recent) > 20 {
		recent = recent[:20]
	}

	response.OK(c, gin.H{
		"company": company,
		"stats": gin.H{
			"project_count":    len(projects),
			"active_projects":  activeProjects,
			"contact_count":    len(contacts),
			"follow_up_count":  len(followUps),
			"estimated_amount": totalAmount,
			"won_amount":       wonAmount,
		},
		"projects":          projSummaries,
		"contacts":          contactSummaries,
		"recent_follow_ups": recent,
	})
}

// ---------- Contacts ----------

func (h *CRMHandler) ListContacts(c *gin.Context) {
	companyID := c.Query("company_id")
	q := c.Query("q")

	type contactRow struct {
		models.Contact
		CompanyName string `json:"company_name"`
	}
	tx := h.DB.Table("contacts").
		Select("contacts.*, COALESCE(companies.name, '') as company_name").
		Joins("LEFT JOIN companies ON companies.id = contacts.company_id")
	if companyID != "" {
		tx = tx.Where("contacts.company_id = ?", companyID)
	}
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("contacts.name LIKE ? OR contacts.phone LIKE ? OR contacts.title LIKE ? OR companies.name LIKE ?", like, like, like, like)
	}
	var list []contactRow
	if err := tx.Order("contacts.updated_at DESC").Scan(&list).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "查询联系人失败")
		return
	}
	response.OK(c, list)
}

func (h *CRMHandler) CreateContact(c *gin.Context) {
	var item models.Contact
	if err := c.ShouldBindJSON(&item); err != nil || item.Name == "" || item.CompanyID == 0 {
		response.Error(c, http.StatusBadRequest, "联系人姓名与所属公司不能为空")
		return
	}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建联系人失败")
		return
	}
	response.OK(c, item)
}

func (h *CRMHandler) UpdateContact(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的联系人 ID")
		return
	}
	var existing models.Contact
	if err := h.DB.First(&existing, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "联系人不存在")
		return
	}
	var payload models.Contact
	if err := c.ShouldBindJSON(&payload); err != nil || payload.Name == "" {
		response.Error(c, http.StatusBadRequest, "联系人姓名不能为空")
		return
	}
	existing.Name = payload.Name
	existing.Title = payload.Title
	existing.Phone = payload.Phone
	existing.Email = payload.Email
	existing.Wechat = payload.Wechat
	existing.Notes = payload.Notes
	if payload.CompanyID != 0 {
		existing.CompanyID = payload.CompanyID
	}
	if err := h.DB.Save(&existing).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "更新联系人失败")
		return
	}
	response.OK(c, existing)
}

func (h *CRMHandler) DeleteContact(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的联系人 ID")
		return
	}
	h.DB.Transaction(func(tx *gorm.DB) error {
		tx.Where("contact_id = ?", id).Delete(&models.ProjectContact{})
		tx.Model(&models.FollowUp{}).Where("contact_id = ?", id).Update("contact_id", nil)
		return tx.Delete(&models.Contact{}, id).Error
	})
	response.OK(c, gin.H{"deleted": true})
}

// ---------- Projects ----------

func (h *CRMHandler) ListProjects(c *gin.Context) {
	companyID := c.Query("company_id")
	stage := c.Query("stage")
	q := c.Query("q")

	type projectRow struct {
		models.Project
		CompanyName string `json:"company_name"`
	}
	tx := h.DB.Table("projects").
		Select("projects.*, COALESCE(companies.name, '') as company_name").
		Joins("LEFT JOIN companies ON companies.id = projects.company_id")
	if companyID != "" {
		tx = tx.Where("projects.company_id = ?", companyID)
	}
	if stage != "" {
		tx = tx.Where("projects.stage = ?", stage)
	}
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("projects.name LIKE ? OR companies.name LIKE ?", like, like)
	}
	var list []projectRow
	if err := tx.Order("projects.updated_at DESC").Scan(&list).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "查询项目列表失败")
		return
	}
	response.OK(c, list)
}

func (h *CRMHandler) CreateProject(c *gin.Context) {
	var item models.Project
	if err := c.ShouldBindJSON(&item); err != nil || item.Name == "" || item.CompanyID == 0 {
		response.Error(c, http.StatusBadRequest, "项目名称与所属公司不能为空")
		return
	}
	if item.Stage == "" {
		item.Stage = "initial"
	}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建项目失败")
		return
	}
	response.OK(c, item)
}

func (h *CRMHandler) UpdateProject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的项目 ID")
		return
	}
	var existing models.Project
	if err := h.DB.First(&existing, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "项目不存在")
		return
	}
	var payload models.Project
	if err := c.ShouldBindJSON(&payload); err != nil || payload.Name == "" {
		response.Error(c, http.StatusBadRequest, "项目名称不能为空")
		return
	}
	existing.Name = payload.Name
	if payload.Stage != "" {
		existing.Stage = payload.Stage
	}
	existing.EstimatedAmount = payload.EstimatedAmount
	existing.TargetDate = payload.TargetDate
	existing.Owner = payload.Owner
	existing.Description = payload.Description
	if err := h.DB.Save(&existing).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "更新项目失败")
		return
	}
	response.OK(c, existing)
}

func (h *CRMHandler) DeleteProject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的项目 ID")
		return
	}
	h.DB.Transaction(func(tx *gorm.DB) error {
		tx.Where("project_id = ?", id).Delete(&models.ProjectContact{})
		tx.Model(&models.FollowUp{}).Where("project_id = ?", id).Update("project_id", nil)
		return tx.Delete(&models.Project{}, id).Error
	})
	response.OK(c, gin.H{"deleted": true})
}

type assignContactReq struct {
	ContactID uint   `json:"contact_id" binding:"required"`
	Role      string `json:"role"`
	Attitude  string `json:"attitude"`
	Notes     string `json:"notes"`
}

func (h *CRMHandler) UpsertProjectContact(c *gin.Context) {
	projID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的项目 ID")
		return
	}
	var req assignContactReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请选择联系人")
		return
	}
	if req.Attitude == "" {
		req.Attitude = "neutral"
	}

	link := models.ProjectContact{
		ProjectID: uint(projID),
		ContactID: req.ContactID,
		Role:      req.Role,
		Attitude:  req.Attitude,
		Notes:     req.Notes,
	}
	err = h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "contact_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "attitude", "notes"}),
	}).Create(&link).Error
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "保存项目人员关联失败")
		return
	}
	response.OK(c, link)
}

func (h *CRMHandler) RemoveProjectContact(c *gin.Context) {
	projID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的项目 ID")
		return
	}
	contactID, err := strconv.Atoi(c.Param("contactId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的联系人 ID")
		return
	}
	h.DB.Where("project_id = ? AND contact_id = ?", projID, contactID).Delete(&models.ProjectContact{})
	response.OK(c, gin.H{"deleted": true})
}

// ---------- FollowUps & Calendar ----------

func (h *CRMHandler) ListFollowUps(c *gin.Context) {
	companyID := c.Query("company_id")
	projectID := c.Query("project_id")

	type fuRow struct {
		models.FollowUp
		CompanyName string `json:"company_name"`
		ProjectName string `json:"project_name"`
		ContactName string `json:"contact_name"`
	}
	tx := h.DB.Table("follow_ups").
		Select("follow_ups.*, COALESCE(companies.name, '') as company_name, COALESCE(projects.name, '') as project_name, COALESCE(contacts.name, '') as contact_name").
		Joins("LEFT JOIN companies ON companies.id = follow_ups.company_id").
		Joins("LEFT JOIN projects ON projects.id = follow_ups.project_id").
		Joins("LEFT JOIN contacts ON contacts.id = follow_ups.contact_id")
	if companyID != "" {
		tx = tx.Where("follow_ups.company_id = ?", companyID)
	}
	if projectID != "" {
		tx = tx.Where("follow_ups.project_id = ?", projectID)
	}
	var list []fuRow
	if err := tx.Order("follow_ups.occurred_at DESC").Limit(200).Scan(&list).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "查询跟进记录失败")
		return
	}
	response.OK(c, list)
}

func (h *CRMHandler) CreateFollowUp(c *gin.Context) {
	var item models.FollowUp
	if err := c.ShouldBindJSON(&item); err != nil || item.CompanyID == 0 || item.Summary == "" {
		response.Error(c, http.StatusBadRequest, "所属公司与跟进纪要不能为空")
		return
	}
	if item.FollowType == "" {
		item.FollowType = "visit"
	}
	if item.OccurredAt.IsZero() {
		item.OccurredAt = time.Now()
	}
	if item.CreatedBy == "" {
		if u, ok := c.Get("username"); ok {
			item.CreatedBy, _ = u.(string)
		}
	}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建跟进记录失败")
		return
	}
	response.OK(c, item)
}

func (h *CRMHandler) UpdateFollowUp(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的跟进 ID")
		return
	}
	var existing models.FollowUp
	if err := h.DB.First(&existing, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "跟进记录不存在")
		return
	}
	var payload models.FollowUp
	if err := c.ShouldBindJSON(&payload); err != nil || payload.Summary == "" {
		response.Error(c, http.StatusBadRequest, "跟进纪要不能为空")
		return
	}
	existing.ProjectID = payload.ProjectID
	existing.ContactID = payload.ContactID
	if payload.FollowType != "" {
		existing.FollowType = payload.FollowType
	}
	if !payload.OccurredAt.IsZero() {
		existing.OccurredAt = payload.OccurredAt
	}
	existing.NextFollowAt = payload.NextFollowAt
	existing.NextFollowNote = payload.NextFollowNote
	existing.Summary = payload.Summary

	if err := h.DB.Save(&existing).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "更新跟进记录失败")
		return
	}
	response.OK(c, existing)
}

func (h *CRMHandler) DeleteFollowUp(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的跟进 ID")
		return
	}
	h.DB.Delete(&models.FollowUp{}, id)
	response.OK(c, gin.H{"deleted": true})
}

func parseTimeParam(val string, fallback time.Time) time.Time {
	if val == "" {
		return fallback
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"}
	for _, l := range layouts {
		if t, err := time.Parse(l, val); err == nil {
			return t
		}
	}
	return fallback
}

// CalendarFollowUps: GET /api/v1/calendar/follow-ups?start=...&end=...
func (h *CRMHandler) CalendarFollowUps(c *gin.Context) {
	now := time.Now()
	start := parseTimeParam(c.Query("start"), now.AddDate(0, -1, 0))
	end := parseTimeParam(c.Query("end"), now.AddDate(0, 1, 0))

	type fuJoin struct {
		models.FollowUp
		CompanyName string `json:"company_name"`
		ProjectName string `json:"project_name"`
		ContactName string `json:"contact_name"`
	}

	var completed []fuJoin
	h.DB.Table("follow_ups").
		Select("follow_ups.*, COALESCE(companies.name, '') as company_name, COALESCE(projects.name, '') as project_name, COALESCE(contacts.name, '') as contact_name").
		Joins("LEFT JOIN companies ON companies.id = follow_ups.company_id").
		Joins("LEFT JOIN projects ON projects.id = follow_ups.project_id").
		Joins("LEFT JOIN contacts ON contacts.id = follow_ups.contact_id").
		Where("follow_ups.occurred_at >= ? AND follow_ups.occurred_at <= ?", start, end).
		Order("follow_ups.occurred_at ASC").
		Scan(&completed)

	var upcoming []fuJoin
	h.DB.Table("follow_ups").
		Select("follow_ups.*, COALESCE(companies.name, '') as company_name, COALESCE(projects.name, '') as project_name, COALESCE(contacts.name, '') as contact_name").
		Joins("LEFT JOIN companies ON companies.id = follow_ups.company_id").
		Joins("LEFT JOIN projects ON projects.id = follow_ups.project_id").
		Joins("LEFT JOIN contacts ON contacts.id = follow_ups.contact_id").
		Where("follow_ups.next_follow_at IS NOT NULL AND follow_ups.next_follow_at >= ? AND follow_ups.next_follow_at <= ?", start, end).
		Order("follow_ups.next_follow_at ASC").
		Scan(&upcoming)

	if completed == nil {
		completed = []fuJoin{}
	}
	if upcoming == nil {
		upcoming = []fuJoin{}
	}

	response.OK(c, gin.H{
		"completed": completed,
		"upcoming":  upcoming,
	})
}
