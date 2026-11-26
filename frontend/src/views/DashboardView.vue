<template>
  <div class="dash-wrap">
    <div class="dash-top-actions">
      <div class="dash-title">业务概览</div>
      <el-dropdown @command="exportCsv">
        <el-button size="small" plain>导出台账 CSV</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="projects">导出项目商机</el-dropdown-item>
            <el-dropdown-item command="companies">导出客户公司</el-dropdown-item>
            <el-dropdown-item command="follow_ups">导出跟进明细</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <el-row :gutter="12" class="stat-row">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="kpi-primary">
          <div class="stat-label">有效商机金额</div>
          <div class="stat-val primary num-tabular">¥ {{ fmtMoney(counts.estimated_amount) }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="kpi-won">
          <div class="stat-label">已赢单金额</div>
          <div class="stat-val won num-tabular">¥ {{ fmtMoney(counts.won_amount) }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6" :md="3">
        <el-card shadow="never"><div class="stat-label">客户公司</div><div class="stat-val sub num-tabular">{{ counts.companies }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="6" :md="3">
        <el-card shadow="never"><div class="stat-label">项目商机</div><div class="stat-val sub num-tabular">{{ counts.projects }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="6" :md="3">
        <el-card shadow="never"><div class="stat-label">联系人</div><div class="stat-val sub num-tabular">{{ counts.contacts }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="6" :md="3">
        <el-card shadow="never"><div class="stat-label">累计跟进</div><div class="stat-val sub num-tabular">{{ counts.follow_ups }}</div></el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" style="margin-top: 14px;" v-if="stageStats.length">
      <template #header>
        <div class="card-hd">
          <span>商机阶段分布</span>
          <el-button type="primary" link @click="router.push('/projects')">进入项目看板</el-button>
        </div>
      </template>
      <div class="funnel-bar">
        <div
          v-for="st in activeFunnel"
          :key="st.stage"
          class="funnel-seg"
          :style="{ width: st.pct + '%', backgroundColor: STAGE_COLORS[st.stage] }"
          :title="`${PROJECT_STAGES[st.stage]?.label}: ${st.count}个 / ¥${fmtMoney(st.amount)}`"
        />
      </div>
      <div class="funnel-legend">
        <div v-for="st in stageStats" :key="st.stage" class="legend-item">
          <span class="dot" :style="{ backgroundColor: STAGE_COLORS[st.stage] }"></span>
          <span class="lg-name">{{ PROJECT_STAGES[st.stage]?.label || st.stage }}</span>
          <span class="lg-num num-tabular">{{ st.count }}单 · ¥{{ fmtMoney(st.amount) }}</span>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" style="margin-top: 14px;">
      <template #header>
        <div class="card-hd">
          <span>近期跟进待办计划</span>
          <el-button type="primary" link @click="router.push('/calendar')">去日历视图</el-button>
        </div>
      </template>

      <el-table class="desktop-only" :data="upcoming" stripe empty-text="暂无待跟进计划">
        <el-table-column label="计划时间" width="195">
          <template #default="{ row }">
            <div class="date-cell">
              <span class="num-tabular">{{ fmtDate(row.next_follow_at) }}</span>
              <el-tag v-if="dueStatus(row.next_follow_at)" size="small" :type="dueStatus(row.next_follow_at).type">
                {{ dueStatus(row.next_follow_at).label }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="company_name" label="客户公司" min-width="150" />
        <el-table-column prop="project_name" label="关联项目" min-width="140">
          <template #default="{ row }">{{ row.project_name || "-" }}</template>
        </el-table-column>
        <el-table-column prop="contact_name" label="联系人" width="110">
          <template #default="{ row }">{{ row.contact_name || "-" }}</template>
        </el-table-column>
        <el-table-column label="待办说明" min-width="220">
          <template #default="{ row }">{{ row.next_follow_note || row.summary }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openCompany(row.company_id)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="mobile-only">
        <div v-if="!upcoming.length" class="empty-tip">暂无待跟进计划</div>
        <div v-for="row in upcoming" :key="row.id" class="m-card" @click="openCompany(row.company_id)">
          <div class="m-card-top">
            <span class="m-company">{{ row.company_name }}</span>
            <el-tag v-if="dueStatus(row.next_follow_at)" size="small" :type="dueStatus(row.next_follow_at).type">
              {{ dueStatus(row.next_follow_at).label }}
            </el-tag>
          </div>
          <div class="m-sub num-tabular">{{ fmtDate(row.next_follow_at) }} · {{ row.project_name || "通用跟进" }}</div>
          <div class="m-note">{{ row.next_follow_note || row.summary }}</div>
        </div>
      </div>
    </el-card>

    <CompanyChainDrawer ref="drawerRef" @changed="loadData" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { ElMessage } from "element-plus"
import client from "../api/client"
import { PROJECT_STAGES, fmtDate, fmtMoney } from "../utils/constants"
import CompanyChainDrawer from "../components/CompanyChainDrawer.vue"

const STAGE_COLORS = {
  initial: "#64748b",
  testing: "#f59e0b",
  quoting: "#3b5bdb",
  won: "#10b981",
  lost: "#ef4444",
}

const router = useRouter()
const drawerRef = ref(null)
const counts = ref({ companies: 0, projects: 0, contacts: 0, follow_ups: 0, estimated_amount: 0, won_amount: 0 })
const upcoming = ref([])
const stageStats = ref([])

const activeFunnel = computed(() => {
  const total = stageStats.value.reduce((acc, s) => acc + (s.count || 0), 0)
  if (!total) return []
  return stageStats.value.filter((s) => s.count > 0).map((s) => ({ ...s, pct: Math.max(4, Math.round((s.count / total) * 100)) }))
})

function dueStatus(val) {
  if (!val) return null
  const d = new Date(val)
  const now = new Date()
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const tomorrowStart = new Date(todayStart.getTime() + 86400000)
  if (d < todayStart) return { label: "已逾期", type: "danger" }
  if (d < tomorrowStart) return { label: "今日", type: "warning" }
  return null
}

async function exportCsv(type) {
  const token = localStorage.getItem("crm_token")
  const resp = await fetch(`/api/v1/export/csv?type=${encodeURIComponent(type)}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!resp.ok) {
    ElMessage.error("导出失败")
    return
  }
  const blob = await resp.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = `crm-${type}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

async function loadData() {
  const data = await client.get("/dashboard")
  counts.value = data.counts || counts.value
  upcoming.value = data.upcoming || []
  stageStats.value = data.stage_stats || []
}

function openCompany(id) {
  drawerRef.value?.open(id)
}

onMounted(loadData)
</script>

<style scoped>
.dash-top-actions { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.dash-title { font-size: 18px; font-weight: 700; color: #0f172a; }
.stat-row :deep(.el-card__body) { padding: 14px 16px; }
.kpi-primary { border-left: 4px solid #3b5bdb !important; }
.kpi-won { border-left: 4px solid #10b981 !important; }
.stat-label { font-size: 12px; color: #64748b; margin-bottom: 6px; }
.stat-val { font-size: 22px; font-weight: 700; color: #0f172a; }
.stat-val.primary { color: #3b5bdb; }
.stat-val.won { color: #059669; }
.stat-val.sub { font-size: 20px; }
.card-hd { display: flex; justify-content: space-between; align-items: center; font-weight: 600; }
.funnel-bar { display: flex; height: 10px; border-radius: 6px; overflow: hidden; background: #e2e8f0; gap: 2px; }
.funnel-seg { height: 100%; transition: width 0.3s; }
.funnel-legend { display: flex; flex-wrap: wrap; gap: 16px; margin-top: 12px; }
.legend-item { display: flex; align-items: center; gap: 6px; font-size: 13px; }
.dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.lg-name { color: #334155; font-weight: 500; }
.lg-num { color: #64748b; }
.date-cell { display: flex; align-items: center; gap: 6px; }
.m-card { padding: 12px; border: 1px solid #e2e8f0; border-radius: 10px; margin-bottom: 10px; background: #fff; }
.m-card-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.m-company { font-weight: 600; color: #1e293b; }
.m-sub { font-size: 12px; color: #64748b; margin-bottom: 4px; }
.m-note { font-size: 13px; color: #334155; }
.empty-tip { text-align: center; color: #94a3b8; padding: 16px; font-size: 13px; }
@media (max-width: 767px) {
  .stat-row .el-col { margin-bottom: 10px; }
}
</style>
