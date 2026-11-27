<template>
  <div>
    <el-card shadow="never">
      <template #header>
        <div class="cal-hd">
          <span>跟进日历与待办排期</span>
          <div class="legend">
            <span class="dot done"></span> 已完成跟进
            <span class="dot todo"></span> 待办跟进计划
          </div>
        </div>
      </template>
      <FullCalendar ref="calRef" :options="calendarOptions" />
    </el-card>

    <el-dialog v-model="detailDlg.visible" title="跟进详情" width="460px">
      <div v-if="detailDlg.item" class="det-body">
        <p><strong>类型：</strong>{{ detailDlg.kind === 'upcoming' ? '待办跟进计划' : '已完成跟进' }}</p>
        <p><strong>客户公司：</strong>{{ detailDlg.item.company_name }}</p>
        <p v-if="detailDlg.item.project_name"><strong>关联项目：</strong>{{ detailDlg.item.project_name }}</p>
        <p v-if="detailDlg.item.contact_name"><strong>联系人：</strong>{{ detailDlg.item.contact_name }}</p>
        <p><strong>方式：</strong>{{ FOLLOW_TYPES[detailDlg.item.follow_type] || detailDlg.item.follow_type }}</p>
        <p><strong>发生时间：</strong>{{ fmtDate(detailDlg.item.occurred_at) }}</p>
        <p v-if="detailDlg.item.next_follow_at"><strong>计划时间：</strong>{{ fmtDate(detailDlg.item.next_follow_at) }}</p>
        <p><strong>纪要/备注：</strong>{{ detailDlg.kind === 'upcoming' ? (detailDlg.item.next_follow_note || detailDlg.item.summary) : detailDlg.item.summary }}</p>
      </div>
      <template #footer>
        <el-button @click="detailDlg.visible = false">关闭</el-button>
        <el-button v-if="detailDlg.kind === 'upcoming'" type="warning" @click="cancelTodo">取消待办</el-button>
        <el-button type="primary" @click="goCompany">打开公司链式详情</el-button>
      </template>
    </el-dialog>

    <CompanyChainDrawer ref="drawerRef" @changed="reloadEvents" />
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import FullCalendar from '@fullcalendar/vue3'
import dayGridPlugin from '@fullcalendar/daygrid'
import timeGridPlugin from '@fullcalendar/timegrid'
import interactionPlugin from '@fullcalendar/interaction'
import client from '../api/client'
import { ElMessage } from 'element-plus'
import { FOLLOW_TYPES, fmtDate } from '../utils/constants'
import CompanyChainDrawer from '../components/CompanyChainDrawer.vue'

const calRef = ref(null)
const drawerRef = ref(null)
const detailDlg = reactive({ visible: false, kind: 'completed', item: null })

const calendarOptions = reactive({
  plugins: [dayGridPlugin, timeGridPlugin, interactionPlugin],
  initialView: 'dayGridMonth',
  locale: 'zh-cn',
  headerToolbar: {
    left: 'prev,next today',
    center: 'title',
    right: 'dayGridMonth,timeGridWeek,timeGridDay'
  },
  buttonText: {
    today: '今天',
    month: '月',
    week: '周',
    day: '日'
  },
  height: 'auto',
  events: async (info, successCallback, failureCallback) => {
    try {
      const data = await client.get('/calendar/follow-ups', {
        params: { start: info.startStr, end: info.endStr }
      })
      const events = []
      for (const item of data.completed || []) {
        events.push({
          id: `c-${item.id}`,
          title: `[已跟进] ${item.company_name}${item.project_name ? ' · ' + item.project_name : ''}`,
          start: item.occurred_at,
          backgroundColor: '#16a34a',
          borderColor: '#16a34a',
          extendedProps: { kind: 'completed', item }
        })
      }
      for (const item of data.upcoming || []) {
        events.push({
          id: `u-${item.id}`,
          title: `[待办] ${item.company_name}${item.next_follow_note ? ' · ' + item.next_follow_note : ''}`,
          start: item.next_follow_at,
          backgroundColor: '#ea580c',
          borderColor: '#ea580c',
          extendedProps: { kind: 'upcoming', item }
        })
      }
      successCallback(events)
    } catch (e) {
      failureCallback(e)
    }
  },
  eventClick: (info) => {
    const { kind, item } = info.event.extendedProps
    detailDlg.kind = kind
    detailDlg.item = item
    detailDlg.visible = true
  }
})

async function cancelTodo() {
  const item = detailDlg.item
  if (!item) return
  try {
    await client.put(`/follow-ups/${item.id}`, {
      project_id: item.project_id || null,
      contact_id: item.contact_id || null,
      follow_type: item.follow_type,
      occurred_at: item.occurred_at,
      next_follow_at: null,
      next_follow_note: '',
      summary: item.summary || ''
    })
    ElMessage.success('已取消待办')
    detailDlg.visible = false
    reloadEvents()
  } catch (e) {}
}

function goCompany() {
  const id = detailDlg.item?.company_id
  detailDlg.visible = false
  if (id) drawerRef.value?.open(id)
}

function reloadEvents() {
  calRef.value?.getApi()?.refetchEvents()
}
</script>

<style scoped>
.cal-hd { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; }
.legend { display: flex; align-items: center; gap: 12px; font-size: 13px; color: #4b5563; }
.dot { display: inline-block; width: 10px; height: 10px; border-radius: 50%; margin-right: 4px; }
.dot.done { background: #16a34a; }
.dot.todo { background: #ea580c; }
.det-body p { margin: 6px 0; font-size: 14px; line-height: 1.5; }
</style>
