<template>
  <div>
    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="q" placeholder="搜索姓名 / 电话 / 职位 / 公司" clearable style="max-width: 280px;" @keyup.enter="load" />
        <el-button @click="load">查询</el-button>
      </div>

      <el-table :data="list" v-loading="loading" stripe style="margin-top: 12px;">
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="company_name" label="所属公司" min-width="160">
          <template #default="{ row }">
            <el-button type="primary" link @click="drawerRef?.open(row.company_id)">{{ row.company_name }}</el-button>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="职位" width="130" />
        <el-table-column prop="phone" label="电话" width="140" />
        <el-table-column prop="wechat" label="微信" width="130" />
        <el-table-column prop="email" label="邮箱" min-width="160" />
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="drawerRef?.open(row.company_id)">查看反查</el-button>
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
import CompanyChainDrawer from '../components/CompanyChainDrawer.vue'

const q = ref('')
const loading = ref(false)
const list = ref([])
const drawerRef = ref(null)

async function load() {
  loading.value = true
  try {
    list.value = await client.get('/contacts', { params: { q: q.value } })
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; flex-wrap: wrap; align-items: center; }
</style>
