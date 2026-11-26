<template>
  <div>
    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="q" placeholder="搜索项目或公司名称" clearable style="max-width: 240px;" @keyup.enter="load" />
        <el-select v-model="stage" placeholder="全部阶段" clearable style="width: 130px;" @change="load" v-if="viewMode === 'list'">
          <el-option v-for="(v, k) in PROJECT_STAGES" :key="k" :label="v.label" :value="k" />
        </el-select>
        <el-button @click="load">查询</el-button>
        <el-radio-group v-model="viewMode" size="small" class="desktop-only" style="margin-left: auto;">
          <el-radio-button value="kanban">看板视图</el-radio-button>
          <el-radio-button value="list">列表视图</el-radio-button>
        </el-radio-group>
      </div>

      <div v-if="viewMode === 'kanban'" class="kanban-board desktop-only" v-loading="loading">
        <div
          v-for="(meta, sKey) in PROJECT_STAGES"
          :key="sKey"
          class="kanban-col"
          @dragover.prevent
          @drop="onDrop($event, sKey)"
        >
          <div class="col-header">
            <div class="col-title">
              <el-tag size="small" :type="meta.type">{{ meta.label }}</el-tag>
              <span class="col-cnt num-tabular">{{ byStage(sKey).length }}</span>
            </div>
            <div class="col-sum num-tabular">¥{{ fmtMoney(stageTotal(sKey)) }}</div>
          </div>
          <div class="col-body">
            <div
              v-for="item in byStage(sKey)"
              :key="item.id"
              class="k-card"
              draggable="true"
              @dragstart="onDragStart($event, item)"
            >
              <div class="k-name" @click="drawerRef?.open(item.company_id)">{{ item.name }}</div>
              <div class="k-co">{{ item.company_name }}</div>
              <div class="k-foot">
                <span class="k-amt num-tabular">¥{{ fmtMoney(item.estimated_amount) }}</span>
                <span class="k-owner">{{ item.owner || "-" }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <el-table v-else class="desktop-only" :data="list" v-loading="loading" stripe style="margin-top: 12px;">
        <el-table-column prop="name" label="项目名称" min-width="180" />
        <el-table-column prop="company_name" label="所属公司" min-width="160">
          <template #default="{ row }">
            <el-button type="primary" link @click="drawerRef?.open(row.company_id)">{{ row.company_name }}</el-button>
          </template>
        </el-table-column>
        <el-table-column label="阶段" width="120">
          <template #default="{ row }">
            <el-select :model-value="row.stage" size="small" @change="(val) => changeStage(row, val)">
              <el-option v-for="(v, k) in PROJECT_STAGES" :key="k" :label="v.label" :value="k" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="预估金额" width="130">
          <template #default="{ row }"><span class="num-tabular">¥{{ fmtMoney(row.estimated_amount) }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="drawerRef?.open(row.company_id)">穿透详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="mobile-only" style="margin-top: 12px;">
        <div v-for="row in list" :key="row.id" class="m-card" @click="drawerRef?.open(row.company_id)">
          <div class="m-top">
            <span class="m-name">{{ row.name }}</span>
            <el-tag size="small" :type="PROJECT_STAGES[row.stage]?.type || 'info'">{{ PROJECT_STAGES[row.stage]?.label || row.stage }}</el-tag>
          </div>
          <div class="m-sub">{{ row.company_name }} · <span class="num-tabular">¥{{ fmtMoney(row.estimated_amount) }}</span></div>
        </div>
      </div>
    </el-card>

    <CompanyChainDrawer ref="drawerRef" @changed="load" />
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue"
import { ElMessage } from "element-plus"
import client from "../api/client"
import { PROJECT_STAGES, fmtMoney } from "../utils/constants"
import CompanyChainDrawer from "../components/CompanyChainDrawer.vue"

const q = ref("")
const stage = ref("")
const viewMode = ref("kanban")
const loading = ref(false)
const list = ref([])
const drawerRef = ref(null)
let draggedItem = null

function byStage(sKey) {
  return list.value.filter((x) => x.stage === sKey)
}

function stageTotal(sKey) {
  return byStage(sKey).reduce((acc, x) => acc + Number(x.estimated_amount || 0), 0)
}

function onDragStart(_ev, item) {
  draggedItem = item
}

async function onDrop(_ev, targetStage) {
  if (!draggedItem || draggedItem.stage === targetStage) return
  await changeStage(draggedItem, targetStage)
  draggedItem = null
}

async function changeStage(item, targetStage) {
  await client.put(`/projects/${item.id}`, { ...item, stage: targetStage })
  item.stage = targetStage
  ElMessage.success(`已调整为「${PROJECT_STAGES[targetStage]?.label || targetStage}」`)
}

async function load() {
  loading.value = true
  try {
    list.value = await client.get("/projects", { params: { q: q.value, stage: viewMode.value === "kanban" ? "" : stage.value } })
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; flex-wrap: wrap; align-items: center; }
.kanban-board { display: grid; grid-template-columns: repeat(5, minmax(200px, 1fr)); gap: 12px; margin-top: 14px; overflow-x: auto; padding-bottom: 6px; }
.kanban-col { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 10px; padding: 10px; min-height: 360px; }
.col-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; padding-bottom: 8px; border-bottom: 1px solid #e2e8f0; }
.col-title { display: flex; align-items: center; gap: 6px; }
.col-cnt { font-size: 12px; color: #64748b; font-weight: 600; }
.col-sum { font-size: 12px; font-weight: 600; color: #334155; }
.k-card { background: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; padding: 10px; margin-bottom: 8px; cursor: grab; transition: box-shadow 0.15s; }
.k-card:hover { box-shadow: 0 2px 6px rgba(15, 23, 42, 0.08); }
.k-name { font-weight: 600; font-size: 14px; color: #1e293b; cursor: pointer; margin-bottom: 4px; }
.k-name:hover { color: #3b5bdb; }
.k-co { font-size: 12px; color: #64748b; margin-bottom: 8px; }
.k-foot { display: flex; justify-content: space-between; align-items: center; font-size: 12px; }
.k-amt { font-weight: 600; color: #3b5bdb; }
.k-owner { color: #94a3b8; }
.m-card { padding: 12px; border: 1px solid #e2e8f0; border-radius: 10px; margin-bottom: 10px; background: #fff; }
.m-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.m-name { font-weight: 600; color: #1e293b; }
.m-sub { font-size: 12px; color: #64748b; }
</style>
