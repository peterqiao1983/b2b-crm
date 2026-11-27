<template>
  <el-drawer v-model="visible" :title="summary?.company?.name || '公司详情'" size="min(920px, 96vw)" destroy-on-close>
    <div v-loading="loading" class="drawer-body" v-if="summary">
      <!-- 公司基础信息与概览指标 -->
      <div class="meta-bar">
        <el-tag :type="COMPANY_STAGES[summary.company.stage]?.type || 'info'">
          {{ COMPANY_STAGES[summary.company.stage]?.label || summary.company.stage }}
        </el-tag>
        <span v-if="summary.company.industry" class="meta-item">行业：{{ summary.company.industry }}</span>
        <span v-if="summary.company.owner" class="meta-item">负责人：{{ summary.company.owner }}</span>
        <span v-if="summary.company.source" class="meta-item">来源：{{ summary.company.source }}</span>
      </div>
      <div v-if="summary.company.notes" class="company-notes">{{ summary.company.notes }}</div>

      <el-row :gutter="10" class="mini-stats">
        <el-col :xs="12" :sm="6"><div class="ms-box"><div class="ms-l">项目总数</div><div class="ms-v">{{ summary.stats.project_count }}（活跃 {{ summary.stats.active_projects }}）</div></div></el-col>
        <el-col :xs="12" :sm="6"><div class="ms-box"><div class="ms-l">累计预估</div><div class="ms-v">{{ fmtMoney(summary.stats.estimated_amount) }}</div></div></el-col>
        <el-col :xs="12" :sm="6"><div class="ms-box"><div class="ms-l">联系人</div><div class="ms-v">{{ summary.stats.contact_count }}</div></div></el-col>
        <el-col :xs="12" :sm="6"><div class="ms-box"><div class="ms-l">跟进记录</div><div class="ms-v">{{ summary.stats.follow_up_count }}</div></div></el-col>
      </el-row>

      <div class="quick-actions">
        <el-button type="primary" size="small" @click="openFollowUpModal(null, null)">+ 记一条跟进</el-button>
        <el-button size="small" @click="openProjectModal(null)">+ 新建项目</el-button>
        <el-button size="small" @click="openContactModal(null)">+ 新建联系人</el-button>
        <el-button size="small" @click="openLinkExistingContact">+ 关联已有联系人</el-button>
      </div>

      <el-tabs v-model="activeTab" style="margin-top: 10px;">
        <!-- Tab 1: 项目层穿透 -->
        <el-tab-pane :label="`项目与专属人员 (${summary.projects.length})`" name="projects">
          <el-empty v-if="summary.projects.length === 0" description="暂无项目，点击上方新建项目" />
          <el-collapse v-else v-model="expandedProjects">
            <el-collapse-item v-for="p in summary.projects" :key="p.id" :name="p.id">
              <template #title>
                <div class="proj-title-row">
                  <span class="proj-name">{{ p.name }}</span>
                  <el-tag size="small" :type="PROJECT_STAGES[p.stage]?.type || 'info'">
                    {{ PROJECT_STAGES[p.stage]?.label || p.stage }}
                  </el-tag>
                  <span class="proj-amt">{{ fmtMoney(p.estimated_amount) }}</span>
                </div>
              </template>

              <div class="proj-ops">
                <el-button size="small" link type="primary" @click="openFollowUpModal(p.id, null)">+ 项目跟进</el-button>
                <el-button size="small" link type="primary" @click="openAssignModal(p)">+ 指派人员</el-button>
                <el-button size="small" link @click="openProjectModal(p)">编辑项目</el-button>
                <el-popconfirm title="确认删除该项目？" @confirm="deleteProject(p.id)">
                  <template #reference>
                    <el-button size="small" link type="danger">删除</el-button>
                  </template>
                </el-popconfirm>
              </div>
              <div v-if="p.description" class="proj-desc">{{ p.description }}</div>

              <div class="sub-sec-title">项目专属人员（{{ p.contacts.length }}）</div>
              <el-table v-if="p.contacts.length" :data="p.contacts" size="small" border style="margin-bottom: 12px;">
                <el-table-column prop="name" label="姓名" width="100" />
                <el-table-column prop="title" label="职位" width="110" />
                <el-table-column prop="role" label="项目角色" width="120" />
                <el-table-column label="态度" width="80">
                  <template #default="{ row }">
                    <el-tag size="small" :type="ATTITUDES[row.attitude]?.type || 'info'">
                      {{ ATTITUDES[row.attitude]?.label || row.attitude }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="phone" label="电话" width="130" />
                <el-table-column label="操作" width="70">
                  <template #default="{ row }">
                    <el-button link type="danger" size="small" @click="removeProjectContact(p.id, row.contact_id)">移除</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <div v-else class="empty-hint">暂未指派人员</div>

            </el-collapse-item>
          </el-collapse>
        </el-tab-pane>

        <!-- Tab 2: 人员层反查 -->
        <el-tab-pane :label="`联系人反查 (${summary.contacts.length})`" name="contacts">
          <el-empty v-if="summary.contacts.length === 0" description="暂无联系人" />
          <div v-else class="contact-list">
            <el-card v-for="ct in summary.contacts" :key="ct.id" shadow="never" class="ct-card">
              <div class="ct-top">
                <div>
                  <strong>{{ ct.name }}</strong>
                  <span v-if="ct.title" class="ct-title">{{ ct.title }}</span>
                </div>
                <div>
                  <el-button link type="primary" size="small" @click="openFollowUpModal(null, ct.id)">+ 记跟进</el-button>
                  <el-button link size="small" @click="openContactModal(ct)">编辑</el-button>
                  <el-popconfirm title="确认删除联系人？" @confirm="deleteContact(ct.id)">
                    <template #reference><el-button link type="danger" size="small">删除</el-button></template>
                  </el-popconfirm>
                </div>
              </div>
              <div class="ct-info">
                <span v-if="ct.phone">电话：{{ ct.phone }}</span>
                <span v-if="ct.wechat">微信：{{ ct.wechat }}</span>
                <span v-if="ct.email">邮箱：{{ ct.email }}</span>
              </div>
              <div class="ct-roles" v-if="ct.projects.length">
                <span>参与项目：</span>
                <el-tag v-for="pr in ct.projects" :key="pr.project_id" size="small" style="margin-right: 6px; margin-bottom: 4px;">
                  {{ pr.project_name }} · {{ pr.role || '成员' }} ({{ ATTITUDES[pr.attitude]?.label || pr.attitude }})
                </el-tag>
              </div>
            </el-card>
          </div>
        </el-tab-pane>

        <!-- Tab 3: 最新 20 条全量动态 -->
        <el-tab-pane :label="`最新动态 (${summary.recent_follow_ups.length})`" name="recent">
          <el-empty v-if="summary.recent_follow_ups.length === 0" description="暂无动态" />
          <el-timeline v-else>
            <el-timeline-item v-for="fu in summary.recent_follow_ups" :key="fu.id" :timestamp="fmtDate(fu.occurred_at)" placement="top">
              <div class="fu-card">
                <div class="fu-head">
                  <el-tag size="small" effect="plain">{{ FOLLOW_TYPES[fu.follow_type] || fu.follow_type }}</el-tag>
                  <span v-if="fu.project_name" class="fu-who">项目：{{ fu.project_name }}</span>
                  <span v-if="fu.contact_name" class="fu-who">联系人：{{ fu.contact_name }}</span>
                  <span class="fu-actions">
                    <el-button link size="small" @click="openEditFollowUp(fu)">编辑</el-button>
                    <el-popconfirm title="确认删除该跟进记录？" @confirm="deleteFollowUp(fu.id)">
                      <template #reference><el-button link type="danger" size="small">删除</el-button></template>
                    </el-popconfirm>
                  </span>
                </div>
                <div class="fu-sum">{{ fu.summary }}</div>
                <div v-if="fu.next_follow_at" class="fu-next">
                  下次计划：{{ fmtDate(fu.next_follow_at) }} {{ fu.next_follow_note ? '· ' + fu.next_follow_note : '' }}
                </div>
              </div>
            </el-timeline-item>
          </el-timeline>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 项目弹窗 -->
    <el-dialog v-model="projDlg.visible" :title="projDlg.form.id ? '编辑项目' : '新建项目'" width="460px" append-to-body>
      <el-form :model="projDlg.form" label-width="90px">
        <el-form-item label="项目名称"><el-input v-model="projDlg.form.name" /></el-form-item>
        <el-form-item label="推进阶段">
          <el-select v-model="projDlg.form.stage" style="width: 100%">
            <el-option v-for="(v, k) in PROJECT_STAGES" :key="k" :label="v.label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="预估金额"><el-input-number v-model="projDlg.form.estimated_amount" :min="0" :step="1000" style="width: 100%" /></el-form-item>
        <el-form-item label="目标节点"><el-date-picker v-model="projDlg.form.target_date" type="date" style="width: 100%" /></el-form-item>
        <el-form-item label="项目说明"><el-input v-model="projDlg.form.description" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="projDlg.visible = false">取消</el-button>
        <el-button type="primary" @click="saveProject">保存</el-button>
      </template>
    </el-dialog>

    <!-- 联系人弹窗 -->
    <el-dialog v-model="ctDlg.visible" :title="ctDlg.form.id ? '编辑联系人' : '新建联系人'" width="460px" append-to-body>
      <el-form :model="ctDlg.form" label-width="80px">
        <el-form-item label="姓名"><el-input v-model="ctDlg.form.name" /></el-form-item>
        <el-form-item label="职位"><el-input v-model="ctDlg.form.title" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="ctDlg.form.phone" /></el-form-item>
        <el-form-item label="微信"><el-input v-model="ctDlg.form.wechat" /></el-form-item>
        <el-form-item label="邮箱"><el-input v-model="ctDlg.form.email" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="ctDlg.form.notes" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ctDlg.visible = false">取消</el-button>
        <el-button type="primary" @click="saveContact">保存</el-button>
      </template>
    </el-dialog>

    <!-- 指派项目人员弹窗 -->
    <el-dialog v-model="assignDlg.visible" title="指派项目人员" width="440px" append-to-body>
      <el-form :model="assignDlg.form" label-width="85px">
        <el-form-item label="联系人">
          <el-select v-model="assignDlg.form.contact_id" style="width: 100%" placeholder="选择公司联系人">
            <el-option v-for="ct in summary?.contacts || []" :key="ct.id" :label="`${ct.name}${ct.title ? ' (' + ct.title + ')' : ''}`" :value="ct.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="项目角色"><el-input v-model="assignDlg.form.role" placeholder="如：决策人 / 技术评估 / 商务接口" /></el-form-item>
        <el-form-item label="态度倾向">
          <el-select v-model="assignDlg.form.attitude" style="width: 100%">
            <el-option v-for="(v, k) in ATTITUDES" :key="k" :label="v.label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="assignDlg.form.notes" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="assignDlg.visible = false">取消</el-button>
        <el-button type="primary" @click="saveAssign">保存</el-button>
      </template>
    </el-dialog>

    <!-- 跟进记录弹窗 -->
    <el-dialog v-model="fuDlg.visible" :title="fuDlg.form.id ? '编辑跟进' : '新增跟进记录'" width="500px" append-to-body>
      <el-form :model="fuDlg.form" label-width="95px">
        <el-form-item label="跟进方式">
          <el-select v-model="fuDlg.form.follow_type" style="width: 100%">
            <el-option v-for="(label, k) in FOLLOW_TYPES" :key="k" :label="label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="关联项目">
          <el-select v-model="fuDlg.form.project_id" clearable style="width: 100%" placeholder="可选">
            <el-option v-for="p in summary?.projects || []" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="对接人">
          <el-select v-model="fuDlg.form.contact_id" clearable style="width: 100%" placeholder="可选" @change="onFuContactChange">
            <el-option v-for="ct in summary?.contacts || []" :key="ct.id" :label="ct.name" :value="ct.id" />
            <el-option label="＋ 新增对接人" value="__new__" />
          </el-select>
        </el-form-item>
        <el-form-item label="发生时间">
          <el-date-picker v-model="fuDlg.form.occurred_at" type="datetime" style="width: 100%" />
        </el-form-item>
        <el-form-item label="跟进纪要">
          <el-input v-model="fuDlg.form.summary" type="textarea" :rows="3" placeholder="记录沟通内容、推进结论..." />
        </el-form-item>
        <el-form-item label="下次跟进">
          <el-date-picker v-model="fuDlg.form.next_follow_at" type="datetime" clearable style="width: 100%" placeholder="可选，将同步到日历待办" />
        </el-form-item>
        <el-form-item label="待办备注">
          <el-input v-model="fuDlg.form.next_follow_note" placeholder="下次要做什么（如：寄送测试样品）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="fuDlg.visible = false">取消</el-button>
        <el-button type="primary" @click="saveFollowUp">保存</el-button>
      </template>
    </el-dialog>

    <!-- 关联已有联系人弹窗 -->
    <el-dialog v-model="linkCtDlg.visible" title="关联已有联系人到本公司" width="460px" append-to-body>
      <el-form label-width="90px">
        <el-form-item label="选择联系人">
          <el-select v-model="linkCtDlg.contactId" filterable placeholder="搜索姓名/电话（优先展示未关联公司）" style="width: 100%">
            <el-option
              v-for="ct in linkCtDlg.candidates"
              :key="ct.id"
              :label="`${ct.name}${ct.phone ? ' (' + ct.phone + ')' : ''}${ct.company_name ? ' - 当前:' + ct.company_name : ' [未关联]'}`"
              :value="ct.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="linkCtDlg.visible = false">取消</el-button>
        <el-button type="primary" @click="saveLinkExistingContact">确认关联</el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import client from '../api/client'
import { ATTITUDES, COMPANY_STAGES, FOLLOW_TYPES, PROJECT_STAGES, fmtDate, fmtMoney } from '../utils/constants'

const emit = defineEmits(['changed'])
const visible = ref(false)
const loading = ref(false)
const companyId = ref(null)
const summary = ref(null)
const activeTab = ref('projects')
const expandedProjects = ref([])

const projDlg = reactive({
  visible: false,
  form: { id: null, name: '', stage: 'initial', estimated_amount: 0, target_date: null, description: '' }
})
const ctDlg = reactive({
  visible: false,
  fromFollowUp: false,
  form: { id: null, name: '', title: '', phone: '', wechat: '', email: '', notes: '' }
})
const assignDlg = reactive({
  visible: false,
  projectId: null,
  form: { contact_id: null, role: '', attitude: 'neutral', notes: '' }
})
const fuDlg = reactive({
  visible: false,
  form: { id: null, project_id: null, contact_id: null, follow_type: 'visit', occurred_at: new Date(), next_follow_at: null, next_follow_note: '', summary: '' }
})
const linkCtDlg = reactive({
  visible: false,
  contactId: null,
  candidates: []
})

async function openLinkExistingContact() {
  const all = await client.get('/contacts')
  linkCtDlg.candidates = all.filter((c) => c.company_id !== companyId.value)
  linkCtDlg.candidates.sort((a, b) => (a.company_id ? 1 : 0) - (b.company_id ? 1 : 0))
  linkCtDlg.contactId = null
  linkCtDlg.visible = true
}

async function saveLinkExistingContact() {
  if (!linkCtDlg.contactId) {
    ElMessage.warning('请选择联系人')
    return
  }
  const target = linkCtDlg.candidates.find((c) => c.id === linkCtDlg.contactId)
  if (!target) return
  await client.put(`/contacts/${target.id}`, {
    ...target,
    company_id: companyId.value
  })
  ElMessage.success('已关联到当前公司')
  linkCtDlg.visible = false
  await refresh()
  emit('changed')
}

async function refresh() {
  if (!companyId.value) return
  loading.value = true
  try {
    summary.value = await client.get(`/companies/${companyId.value}/chain-summary`)
    if (expandedProjects.value.length === 0 && summary.value.projects.length > 0) {
      expandedProjects.value = [summary.value.projects[0].id]
    }
  } finally {
    loading.value = false
  }
}

async function open(id) {
  companyId.value = id
  visible.value = true
  expandedProjects.value = []
  await refresh()
}

function openProjectModal(p) {
  if (p) {
    projDlg.form = { id: p.id, name: p.name, stage: p.stage, estimated_amount: p.estimated_amount, target_date: p.target_date, description: p.description }
  } else {
    projDlg.form = { id: null, name: '', stage: 'initial', estimated_amount: 0, target_date: null, description: '' }
  }
  projDlg.visible = true
}

async function saveProject() {
  const payload = { ...projDlg.form, company_id: companyId.value }
  if (projDlg.form.id) {
    await client.put(`/projects/${projDlg.form.id}`, payload)
  } else {
    await client.post('/projects', payload)
  }
  ElMessage.success('项目已保存')
  projDlg.visible = false
  await refresh()
  emit('changed')
}

async function deleteProject(id) {
  await client.delete(`/projects/${id}`)
  ElMessage.success('已删除项目')
  await refresh()
  emit('changed')
}

function onFuContactChange(val) {
  if (val === '__new__') {
    fuDlg.form.contact_id = null
    ctDlg.fromFollowUp = true
    openContactModal(null)
  }
}

function openContactModal(ct) {
  if (ct) {
    ctDlg.form = { id: ct.id, name: ct.name, title: ct.title, phone: ct.phone, wechat: ct.wechat, email: ct.email, notes: ct.notes }
  } else {
    ctDlg.form = { id: null, name: '', title: '', phone: '', wechat: '', email: '', notes: '' }
  }
  ctDlg.visible = true
}

async function saveContact() {
  const payload = { ...ctDlg.form, company_id: companyId.value }
  let saved = null
  if (ctDlg.form.id) {
    saved = await client.put(`/contacts/${ctDlg.form.id}`, payload)
  } else {
    saved = await client.post('/contacts', payload)
  }
  ElMessage.success('联系人已保存')
  ctDlg.visible = false
  await refresh()
  if (ctDlg.fromFollowUp && saved && saved.id) {
    fuDlg.form.contact_id = saved.id
    ctDlg.fromFollowUp = false
  }
  emit('changed')
}

async function deleteContact(id) {
  await client.delete(`/contacts/${id}`)
  ElMessage.success('已删除联系人')
  await refresh()
  emit('changed')
}

function openAssignModal(p) {
  assignDlg.projectId = p.id
  assignDlg.form = { contact_id: null, role: '', attitude: 'neutral', notes: '' }
  assignDlg.visible = true
}

async function saveAssign() {
  await client.post(`/projects/${assignDlg.projectId}/contacts`, assignDlg.form)
  ElMessage.success('已保存项目人员')
  assignDlg.visible = false
  await refresh()
}

async function removeProjectContact(projId, contactId) {
  await client.delete(`/projects/${projId}/contacts/${contactId}`)
  ElMessage.success('已移除')
  await refresh()
}

function openFollowUpModal(projectId, contactId) {
  fuDlg.form = {
    id: null,
    project_id: projectId,
    contact_id: contactId,
    follow_type: 'visit',
    occurred_at: new Date(),
    next_follow_at: null,
    next_follow_note: '',
    summary: ''
  }
  fuDlg.visible = true
}

function openEditFollowUp(fu) {
  fuDlg.form = {
    id: fu.id,
    project_id: fu.project_id || null,
    contact_id: fu.contact_id || null,
    follow_type: fu.follow_type,
    occurred_at: fu.occurred_at ? new Date(fu.occurred_at) : new Date(),
    next_follow_at: fu.next_follow_at ? new Date(fu.next_follow_at) : null,
    next_follow_note: fu.next_follow_note || '',
    summary: fu.summary
  }
  fuDlg.visible = true
}

async function saveFollowUp() {
  const payload = { ...fuDlg.form, company_id: companyId.value }
  if (fuDlg.form.id) {
    await client.put(`/follow-ups/${fuDlg.form.id}`, payload)
  } else {
    await client.post('/follow-ups', payload)
  }
  ElMessage.success('跟进记录已保存')
  fuDlg.visible = false
  await refresh()
  emit('changed')
}

async function deleteFollowUp(id) {
  await client.delete(`/follow-ups/${id}`)
  ElMessage.success('已删除')
  await refresh()
  emit('changed')
}

defineExpose({ open })
</script>

<style scoped>
.meta-bar { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; margin-bottom: 10px; }
.meta-item { font-size: 13px; color: #4b5563; }
.company-notes { background: #f8fafc; padding: 8px 12px; border-radius: 6px; font-size: 13px; color: #475569; margin-bottom: 12px; }
.mini-stats .el-col { margin-bottom: 8px; }
.ms-box { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; padding: 8px 10px; }
.ms-l { font-size: 12px; color: #64748b; }
.ms-v { font-size: 15px; font-weight: 600; margin-top: 2px; }
.quick-actions { display: flex; gap: 8px; flex-wrap: wrap; margin: 10px 0; }
.proj-title-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.proj-name { font-weight: 600; }
.proj-amt { color: #059669; font-size: 13px; }
.proj-ops { display: flex; gap: 10px; margin-bottom: 8px; }
.proj-desc { font-size: 13px; color: #4b5563; margin-bottom: 10px; }
.sub-sec-title { font-size: 13px; font-weight: 600; color: #374151; margin: 8px 0 6px; }
.empty-hint { font-size: 12px; color: #9ca3af; margin-bottom: 8px; }
.fu-card { background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 6px; padding: 8px 10px; }
.fu-head { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 4px; }
.fu-who { font-size: 12px; color: #6b7280; }
.fu-actions { margin-left: auto; }
.fu-sum { font-size: 13px; white-space: pre-wrap; }
.fu-next { font-size: 12px; color: #d97706; margin-top: 4px; }
.contact-list { display: flex; flex-direction: column; gap: 10px; }
.ct-top { display: flex; justify-content: space-between; align-items: center; }
.ct-title { margin-left: 8px; font-size: 13px; color: #6b7280; }
.ct-info { display: flex; gap: 14px; flex-wrap: wrap; font-size: 13px; color: #4b5563; margin: 6px 0; }
.ct-roles { font-size: 13px; margin-top: 4px; }
.ct-fu-line { font-size: 12px; color: #374151; margin-top: 4px; display: flex; gap: 6px; }
.ct-fu-date { color: #6b7280; flex-shrink: 0; }
</style>
