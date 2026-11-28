<template>
  <div>
    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="q" placeholder="搜索公司名称 / 行业 / 负责人" clearable style="max-width: 260px;" @keyup.enter="load" />
        <el-select v-model="stage" placeholder="全部阶段" clearable style="width: 140px;" @change="load">
          <el-option v-for="(v, k) in COMPANY_STAGES" :key="k" :label="v.label" :value="k" />
        </el-select>
        <el-button @click="load">查询</el-button>
        <el-button type="primary" style="margin-left: auto;" @click="openEdit(null)">+ 新建公司</el-button>
      </div>

      <el-table class="desktop-only" :data="list" v-loading="loading" stripe style="margin-top: 12px;">
        <el-table-column prop="name" label="公司名称" min-width="180">
          <template #default="{ row }">
            <el-button type="primary" link @click="openChain(row.id)">{{ row.name }}</el-button>
          </template>
        </el-table-column>
        <el-table-column label="阶段" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="COMPANY_STAGES[row.stage]?.type || 'info'">
              {{ COMPANY_STAGES[row.stage]?.label || row.stage }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="industry" label="行业" width="120" />
        <el-table-column prop="project_count" label="项目数" width="80" />
        <el-table-column prop="contact_count" label="联系人" width="80" />
        <el-table-column prop="owner" label="负责人" width="100" />
        <el-table-column label="上次跟进" width="150">
          <template #default="{ row }">
            <span class="num-tabular">{{ fmtDate(row.last_follow_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openChain(row.id)">链式详情</el-button>
            <el-button link size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="confirmRemove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="mobile-only" style="margin-top: 12px;">
        <div v-for="row in list" :key="row.id" class="m-card">
          <div class="m-top" @click="openChain(row.id)">
            <span class="m-name">{{ row.name }}</span>
            <el-tag size="small" :type="COMPANY_STAGES[row.stage]?.type || 'info'">{{ COMPANY_STAGES[row.stage]?.label || row.stage }}</el-tag>
          </div>
          <div class="m-sub">{{ row.industry || "未填行业" }} · 项目 {{ row.project_count }} · 联系人 {{ row.contact_count }} · 上次跟进 {{ fmtDate(row.last_follow_at) }}</div>
          <div class="m-acts">
            <el-button link type="primary" size="small" @click="openChain(row.id)">链式详情</el-button>
            <el-button link size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="confirmRemove(row)">删除</el-button>
          </div>
        </div>
      </div>
    </el-card>

    <el-dialog v-model="dlg.visible" :title="dlg.form.id ? '编辑公司' : '新建公司'" width="480px">
      <el-form :model="dlg.form" label-width="80px">
        <el-form-item label="公司名称"><el-input v-model="dlg.form.name" /></el-form-item>
        <el-form-item label="客户阶段">
          <el-select v-model="dlg.form.stage" style="width: 100%">
            <el-option v-for="(v, k) in COMPANY_STAGES" :key="k" :label="v.label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="所属行业"><el-input v-model="dlg.form.industry" /></el-form-item>
        <el-form-item label="负责人"><el-input v-model="dlg.form.owner" /></el-form-item>
        <el-form-item label="线索来源"><el-input v-model="dlg.form.source" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="dlg.form.address" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="dlg.form.notes" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg.visible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <CompanyChainDrawer ref="drawerRef" @changed="load" />
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import client from '../api/client'
import { COMPANY_STAGES, fmtDate } from '../utils/constants'
import CompanyChainDrawer from '../components/CompanyChainDrawer.vue'

const q = ref('')
const stage = ref('')
const loading = ref(false)
const list = ref([])
const drawerRef = ref(null)

const dlg = reactive({
  visible: false,
  form: { id: null, name: '', stage: 'lead', industry: '', owner: 'peter', source: '', address: '', notes: '' }
})

async function load() {
  loading.value = true
  try {
    list.value = await client.get('/companies', { params: { q: q.value, stage: stage.value } })
  } finally {
    loading.value = false
  }
}

function openEdit(row) {
  if (row) {
    dlg.form = { ...row }
  } else {
    dlg.form = { id: null, name: '', stage: 'lead', industry: '', owner: 'peter', source: '', address: '', notes: '' }
  }
  dlg.visible = true
}

async function save() {
  if (dlg.form.id) {
    await client.put(`/companies/${dlg.form.id}`, dlg.form)
  } else {
    await client.post('/companies', dlg.form)
  }
  ElMessage.success('公司已保存')
  dlg.visible = false
  await load()
}

async function confirmRemove(row) {
  try {
    await ElMessageBox.prompt(
      `请输入公司名称「${row.name}」确认移入回收站（将连带移入其项目与跟进记录）：`,
      "高危操作确认",
      {
        confirmButtonText: "确认删除",
        cancelButtonText: "取消",
        inputValidator: (val) => val === row.name || "输入的公司名称不匹配",
      }
    )
    await client.delete(`/companies/${row.id}`)
    ElMessage.success("已移入回收站")
    await load()
  } catch {
    // cancelled
  }
}

function openChain(id) {
  drawerRef.value?.open(id)
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; flex-wrap: wrap; align-items: center; }
.m-card { padding: 12px; border: 1px solid #e2e8f0; border-radius: 10px; margin-bottom: 10px; background: #fff; }
.m-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; cursor: pointer; }
.m-name { font-weight: 600; color: #1e293b; }
.m-sub { font-size: 12px; color: #64748b; margin-bottom: 6px; }
.m-acts { display: flex; gap: 8px; }
</style>
