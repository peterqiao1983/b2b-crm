<template>
  <div>
    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="q" placeholder="搜索姓名 / 电话 / 职位 / 公司" clearable style="max-width: 260px;" @keyup.enter="load" />
        <el-select v-model="unlinked" style="width: 150px;" @change="load">
          <el-option label="全部联系人" value="" />
          <el-option label="仅未关联公司" value="1" />
        </el-select>
        <el-button @click="load">查询</el-button>
        <el-button type="primary" style="margin-left: auto;" @click="openEdit(null)">+ 新建联系人</el-button>
      </div>

      <el-table class="desktop-only" :data="list" v-loading="loading" stripe style="margin-top: 12px;">
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column label="所属公司" min-width="180">
          <template #default="{ row }">
            <el-button v-if="row.company_id" type="primary" link @click="drawerRef?.open(row.company_id)">
              {{ row.company_name || `公司#${row.company_id}` }}
            </el-button>
            <el-tag v-else size="small" type="warning">未关联公司</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="职位" width="130" />
        <el-table-column prop="phone" label="电话" width="140" />
        <el-table-column prop="wechat" label="微信" width="130" />
        <el-table-column prop="email" label="邮箱" min-width="160" />
        <el-table-column label="操作" width="210" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.company_id" link type="primary" size="small" @click="drawerRef?.open(row.company_id)">链式反查</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row)">{{ row.company_id ? '编辑' : '关联公司/编辑' }}</el-button>
            <el-popconfirm title="确认删除该联系人？" @confirm="remove(row.id)">
              <template #reference><el-button link type="danger" size="small">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="mobile-only" style="margin-top: 12px;">
        <div v-for="row in list" :key="row.id" class="m-card">
          <div class="m-top">
            <span class="m-name">{{ row.name }} <small v-if="row.title" style="color:#64748b;font-weight:400;">· {{ row.title }}</small></span>
            <el-button v-if="row.company_id" link type="primary" size="small" @click="drawerRef?.open(row.company_id)">{{ row.company_name || `公司#${row.company_id}` }}</el-button>
          </div>
          <div class="m-acts">
            <a v-if="row.phone" :href="`tel:${row.phone}`" class="tel-link">拨打 {{ row.phone }}</a>
            <el-button v-if="row.wechat" link type="primary" size="small" @click="copyWechat(row.wechat)">复制微信: {{ row.wechat }}</el-button>
            <el-button link size="small" @click="openEdit(row)">编辑</el-button>
          </div>
        </div>
      </div>
    </el-card>

    <el-dialog v-model="dlg.visible" :title="dlg.form.id ? '编辑联系人' : '新建联系人'" width="480px">
      <el-form :model="dlg.form" label-width="85px">
        <el-form-item label="姓名" required><el-input v-model="dlg.form.name" placeholder="必填" /></el-form-item>
        <el-form-item label="所属公司">
          <el-select v-model="dlg.form.company_id" filterable clearable placeholder="可留空，后期再关联" style="width: 100%">
            <el-option v-for="c in companies" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="职位"><el-input v-model="dlg.form.title" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="dlg.form.phone" /></el-form-item>
        <el-form-item label="微信"><el-input v-model="dlg.form.wechat" /></el-form-item>
        <el-form-item label="邮箱"><el-input v-model="dlg.form.email" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="dlg.form.notes" type="textarea" :rows="2" /></el-form-item>
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
import { ElMessage } from 'element-plus'
import client from '../api/client'
import CompanyChainDrawer from '../components/CompanyChainDrawer.vue'

const q = ref('')
const unlinked = ref('')
const loading = ref(false)
const list = ref([])
const companies = ref([])
const drawerRef = ref(null)

const dlg = reactive({
  visible: false,
  form: { id: null, company_id: null, name: '', title: '', phone: '', wechat: '', email: '', notes: '' }
})

async function load() {
  loading.value = true
  try {
    list.value = await client.get('/contacts', { params: { q: q.value, unlinked: unlinked.value } })
  } finally {
    loading.value = false
  }
}

async function loadCompanies() {
  companies.value = await client.get('/companies')
}

async function openEdit(row) {
  await loadCompanies()
  if (row) {
    dlg.form = {
      id: row.id,
      company_id: row.company_id || null,
      name: row.name,
      title: row.title || '',
      phone: row.phone || '',
      wechat: row.wechat || '',
      email: row.email || '',
      notes: row.notes || ''
    }
  } else {
    dlg.form = { id: null, company_id: null, name: '', title: '', phone: '', wechat: '', email: '', notes: '' }
  }
  dlg.visible = true
}

async function save() {
  if (!dlg.form.name) {
    ElMessage.warning('请输入联系人姓名')
    return
  }
  const payload = { ...dlg.form, company_id: dlg.form.company_id || null }
  if (dlg.form.id) {
    await client.put(`/contacts/${dlg.form.id}`, payload)
  } else {
    await client.post('/contacts', payload)
  }
  ElMessage.success('联系人已保存')
  dlg.visible = false
  await load()
}

async function remove(id) {
  await client.delete(`/contacts/${id}`)
  ElMessage.success('已删除')
  await load()
}

async function copyWechat(wx) {
  try {
    await navigator.clipboard.writeText(wx)
    ElMessage.success("微信号已复制")
  } catch {
    ElMessage.info(wx)
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; flex-wrap: wrap; align-items: center; }
.m-card { padding: 12px; border: 1px solid #e2e8f0; border-radius: 10px; margin-bottom: 10px; background: #fff; }
.m-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.m-name { font-weight: 600; color: #1e293b; }
.m-acts { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
.tel-link { font-size: 13px; color: #3b5bdb; text-decoration: none; font-weight: 500; }
</style>
