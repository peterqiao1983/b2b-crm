<template>
  <div>
    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="q" placeholder="搜索项目或公司名称" clearable style="max-width: 260px;" @keyup.enter="load" />
        <el-select v-model="stage" placeholder="全部阶段" clearable style="width: 140px;" @change="load">
          <el-option v-for="(v, k) in PROJECT_STAGES" :key="k" :label="v.label" :value="k" />
        </el-select>
        <el-button @click="load">查询</el-button>
      </div>

      <el-table :data="list" v-loading="loading" stripe style="margin-top: 12px;">
        <el-table-column prop="name" label="项目名称" min-width="180" />
        <el-table-column prop="company_name" label="所属公司" min-width="160">
          <template #default="{ row }">
            <el-button type="primary" link @click="drawerRef?.open(row.company_id)">{{ row.company_name }}</el-button>
          </template>
        </el-table-column>
        <el-table-column label="阶段" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="PROJECT_STAGES[row.stage]?.type || 'info'">
              {{ PROJECT_STAGES[row.stage]?.label || row.stage }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="预估金额" width="130">
          <template #default="{ row }">{{ fmtMoney(row.estimated_amount) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="drawerRef?.open(row.company_id)">穿透详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <CompanyChainDrawer ref="drawerRef" @changed="load" />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import client from '../api/client'
import { PROJECT_STAGES, fmtMoney } from '../utils/constants'
import CompanyChainDrawer from '../components/CompanyChainDrawer.vue'

const q = ref('')
const stage = ref('')
const loading = ref(false)
const list = ref([])
const drawerRef = ref(null)

async function load() {
  loading.value = true
  try {
    list.value = await client.get('/projects', { params: { q: q.value, stage: stage.value } })
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; flex-wrap: wrap; align-items: center; }
</style>
