<template>
  <div class="dash-wrap">
    <el-row :gutter="12" class="stat-row">
      <el-col :xs="12" :sm="8" :md="4">
        <el-card shadow="never"><div class="stat-label">客户公司</div><div class="stat-val">{{ counts.companies }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="4">
        <el-card shadow="never"><div class="stat-label">项目商机</div><div class="stat-val">{{ counts.projects }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="4">
        <el-card shadow="never"><div class="stat-label">联系人</div><div class="stat-val">{{ counts.contacts }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="4">
        <el-card shadow="never"><div class="stat-label">累计跟进</div><div class="stat-val">{{ counts.follow_ups }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="4">
        <el-card shadow="never"><div class="stat-label">有效商机金额</div><div class="stat-val">{{ fmtMoney(counts.estimated_amount) }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="4">
        <el-card shadow="never"><div class="stat-label">已赢单金额</div><div class="stat-val won">{{ fmtMoney(counts.won_amount) }}</div></el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" style="margin-top: 16px;">
      <template #header>
        <div class="card-hd">
          <span>近期跟进待办计划</span>
          <el-button type="primary" link @click="router.push('/calendar')">去日历视图</el-button>
        </div>
      </template>
      <el-table :data="upcoming" stripe empty-text="暂无待跟进计划">
        <el-table-column label="计划时间" width="160">
          <template #default="{ row }">{{ fmtDate(row.next_follow_at) }}</template>
        </el-table-column>
        <el-table-column prop="company_name" label="客户公司" min-width="150" />
        <el-table-column prop="project_name" label="关联项目" min-width="140">
          <template #default="{ row }">{{ row.project_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="contact_name" label="联系人" width="110">
          <template #default="{ row }">{{ row.contact_name || '-' }}</template>
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
    </el-card>

    <CompanyChainDrawer ref="drawerRef" @changed="loadData" />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import client from '../api/client'
import { fmtDate, fmtMoney } from '../utils/constants'
import CompanyChainDrawer from '../components/CompanyChainDrawer.vue'

const router = useRouter()
const drawerRef = ref(null)
const counts = ref({
  companies: 0,
  projects: 0,
  contacts: 0,
  follow_ups: 0,
  estimated_amount: 0,
  won_amount: 0
})
const upcoming = ref([])

async function loadData() {
  const data = await client.get('/dashboard')
  counts.value = data.counts || counts.value
  upcoming.value = data.upcoming || []
}

function openCompany(id) {
  drawerRef.value?.open(id)
}

onMounted(loadData)
</script>

<style scoped>
.stat-row .el-col {
  margin-bottom: 10px;
}
.stat-label {
  font-size: 13px;
  color: #6b7280;
}
.stat-val {
  font-size: 22px;
  font-weight: 700;
  margin-top: 6px;
}
.stat-val.won {
  color: #16a34a;
}
.card-hd {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
