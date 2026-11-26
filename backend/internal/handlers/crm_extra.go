package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"b2b-crm/internal/models"
	"b2b-crm/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func recordAudit(db *gorm.DB, c *gin.Context, action, entityType string, entityID uint, entityName, detail string) {
	actor := "peter"
	if c != nil {
		if u, ok := c.Get("username"); ok {
			if s, ok2 := u.(string); ok2 && s != "" {
				actor = s
			}
		}
	}
	entry := models.AuditLog{
		Actor:      actor,
		Source:     "web",
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		EntityName: entityName,
		Detail:     detail,
		CreatedAt:  time.Now(),
	}
	_ = db.Create(&entry).Error
}

type StageStat struct {
	Stage  string  `json:"stage"`
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
}

func computeStageStats(db *gorm.DB) []StageStat {
	stages := []string{"initial", "testing", "quoting", "won", "lost"}
	byStage := map[string]StageStat{}
	for _, s := range stages {
		byStage[s] = StageStat{Stage: s}
	}
	var rows []StageStat
	db.Model(&models.Project{}).
		Select("stage, COUNT(*) as count, COALESCE(SUM(estimated_amount), 0) as amount").
		Group("stage").
		Scan(&rows)
	for _, r := range rows {
		byStage[r.Stage] = r
	}
	out := make([]StageStat, 0, len(stages))
	for _, s := range stages {
		out = append(out, byStage[s])
	}
	return out
}

func parsePageParams(c *gin.Context) (usePage bool, page, pageSize int) {
	pStr := c.Query("page")
	if pStr == "" {
		return false, 1, 0
	}
	page, err := strconv.Atoi(pStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err = strconv.Atoi(c.Query("page_size"))
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return true, page, pageSize
}

func (h *CRMHandler) ListAuditLogs(c *gin.Context) {
	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	var logs []models.AuditLog
	if err := h.DB.Order("created_at DESC, id DESC").Limit(limit).Find(&logs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "查询操作日志失败")
		return
	}
	response.OK(c, logs)
}

func (h *CRMHandler) ListTrash(c *gin.Context) {
	var companies []models.Company
	var projects []models.Project
	var contacts []models.Contact
	var followUps []models.FollowUp

	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at DESC").Limit(100).Find(&companies)
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at DESC").Limit(100).Find(&projects)
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at DESC").Limit(100).Find(&contacts)
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at DESC").Limit(100).Find(&followUps)

	response.OK(c, gin.H{
		"companies":  companies,
		"projects":   projects,
		"contacts":   contacts,
		"follow_ups": followUps,
	})
}

type restoreReq struct {
	EntityType string `json:"entity_type" binding:"required"`
	EntityID   uint   `json:"entity_id" binding:"required"`
}

func (h *CRMHandler) RestoreTrash(c *gin.Context) {
	var req restoreReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	var name string
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		switch req.EntityType {
		case "company":
			var comp models.Company
			if err := tx.Unscoped().First(&comp, req.EntityID).Error; err != nil {
				return err
			}
			name = comp.Name
			if err := tx.Unscoped().Model(&models.Company{}).Where("id = ?", req.EntityID).Update("deleted_at", nil).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Model(&models.Project{}).Where("company_id = ?", req.EntityID).Update("deleted_at", nil).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Model(&models.Contact{}).Where("company_id = ?", req.EntityID).Update("deleted_at", nil).Error; err != nil {
				return err
			}
			return tx.Unscoped().Model(&models.FollowUp{}).Where("company_id = ?", req.EntityID).Update("deleted_at", nil).Error
		case "project":
			var proj models.Project
			if err := tx.Unscoped().First(&proj, req.EntityID).Error; err != nil {
				return err
			}
			name = proj.Name
			return tx.Unscoped().Model(&models.Project{}).Where("id = ?", req.EntityID).Update("deleted_at", nil).Error
		case "contact":
			var cont models.Contact
			if err := tx.Unscoped().First(&cont, req.EntityID).Error; err != nil {
				return err
			}
			name = cont.Name
			return tx.Unscoped().Model(&models.Contact{}).Where("id = ?", req.EntityID).Update("deleted_at", nil).Error
		case "follow_up":
			var fu models.FollowUp
			if err := tx.Unscoped().First(&fu, req.EntityID).Error; err != nil {
				return err
			}
			name = fu.FollowType
			return tx.Unscoped().Model(&models.FollowUp{}).Where("id = ?", req.EntityID).Update("deleted_at", nil).Error
		default:
			return fmt.Errorf("不支持的实体类型")
		}
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "恢复失败: "+err.Error())
		return
	}
	recordAudit(h.DB, c, "restore", req.EntityType, req.EntityID, name, "从回收站恢复")
	response.OK(c, gin.H{"restored": true})
}

func (h *CRMHandler) ExportCSV(c *gin.Context) {
	exportType := c.DefaultQuery("type", "projects")
	var buf bytes.Buffer
	// UTF-8 BOM for Excel
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)

	switch exportType {
	case "companies":
		_ = w.Write([]string{"ID", "公司名称", "客户阶段", "行业", "负责人", "来源", "地址", "备注", "创建时间"})
		var list []models.Company
		h.DB.Order("id ASC").Find(&list)
		for _, item := range list {
			_ = w.Write([]string{
				strconv.Itoa(int(item.ID)), item.Name, item.Stage, item.Industry,
				item.Owner, item.Source, item.Address, item.Notes, item.CreatedAt.Format("2006-01-02 15:04"),
			})
		}
	case "follow_ups":
		_ = w.Write([]string{"ID", "跟进时间", "客户公司", "关联项目", "联系人", "方式", "跟进内容", "下次计划时间", "下次待办"})
		type row struct {
			models.FollowUp
			CompanyName string
			ProjectName string
			ContactName string
		}
		var rows []row
		h.DB.Table("follow_ups").
			Select("follow_ups.*, COALESCE(companies.name,) as company_name, COALESCE(projects.name,) as project_name, COALESCE(contacts.name,) as contact_name").
			Joins("LEFT JOIN companies ON companies.id = follow_ups.company_id").
			Joins("LEFT JOIN projects ON projects.id = follow_ups.project_id").
			Joins("LEFT JOIN contacts ON contacts.id = follow_ups.contact_id").
			Where("follow_ups.deleted_at IS NULL").
			Order("follow_ups.occurred_at DESC").
			Scan(&rows)
		for _, r := range rows {
			nextAt := ""
			if r.NextFollowAt != nil {
				nextAt = r.NextFollowAt.Format("2006-01-02 15:04")
			}
			_ = w.Write([]string{
				strconv.Itoa(int(r.ID)), r.OccurredAt.Format("2006-01-02 15:04"),
				r.CompanyName, r.ProjectName, r.ContactName, r.FollowType, r.Summary, nextAt, r.NextFollowNote,
			})
		}
	default:
		exportType = "projects"
		_ = w.Write([]string{"ID", "项目名称", "客户公司", "阶段", "预计金额", "预计成交日", "负责人", "描述"})
		type row struct {
			models.Project
			CompanyName string
		}
		var rows []row
		h.DB.Table("projects").
			Select("projects.*, COALESCE(companies.name,) as company_name").
			Joins("LEFT JOIN companies ON companies.id = projects.company_id").
			Where("projects.deleted_at IS NULL").
			Order("projects.id ASC").
			Scan(&rows)
		for _, r := range rows {
			td := ""
			if r.TargetDate != nil {
				td = r.TargetDate.Format("2006-01-02")
			}
			_ = w.Write([]string{
				strconv.Itoa(int(r.ID)), r.Name, r.CompanyName, r.Stage,
				fmt.Sprintf("%.2f", r.EstimatedAmount), td, r.Owner, r.Description,
			})
		}
	}

	w.Flush()
	filename := fmt.Sprintf("crm-%s-%s.csv", exportType, time.Now().Format("20060102"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}
