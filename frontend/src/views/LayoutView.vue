<template>
  <el-container class="layout-root">
    <el-header class="top-header">
      <div class="header-left">
        <el-button class="mobile-only menu-trigger" text @click="mobileNav = true">
          <el-icon :size="20"><Menu /></el-icon>
        </el-button>
        <span class="brand" @click="router.push('/dashboard')">B2B CRM</span>
        <el-menu mode="horizontal" :default-active="route.path" :ellipsis="false" router class="nav-menu desktop-only">
          <el-menu-item index="/dashboard">工作台</el-menu-item>
          <el-menu-item index="/calendar">日历排期</el-menu-item>
          <el-menu-item index="/companies">客户公司</el-menu-item>
          <el-menu-item index="/projects">项目商机</el-menu-item>
          <el-menu-item index="/contacts">联系人</el-menu-item>
        </el-menu>
      </div>
      <div class="header-right">
        <el-button class="desktop-only" size="small" plain @click="openAudit">操作日志</el-button>
        <el-button class="desktop-only" size="small" plain @click="openTrash">回收站</el-button>
        <el-dropdown @command="onCmd">
          <span class="user-btn">
            {{ auth.user?.display_name || auth.user?.username || "用户" }}
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="audit">操作日志</el-dropdown-item>
              <el-dropdown-item command="trash">回收站</el-dropdown-item>
              <el-dropdown-item command="pwd" divided>修改密码</el-dropdown-item>
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>

    <el-main class="main-body">
      <router-view />
    </el-main>

    <el-drawer v-model="mobileNav" direction="ltr" size="220px" title="B2B CRM">
      <el-menu :default-active="route.path" router @select="mobileNav = false" style="border-right: none;">
        <el-menu-item index="/dashboard">工作台</el-menu-item>
        <el-menu-item index="/calendar">日历排期</el-menu-item>
        <el-menu-item index="/companies">客户公司</el-menu-item>
        <el-menu-item index="/projects">项目商机</el-menu-item>
        <el-menu-item index="/contacts">联系人</el-menu-item>
      </el-menu>
    </el-drawer>

    <el-drawer v-model="auditVisible" title="近期操作审计日志" size="min(620px, 94vw)">
      <el-table :data="auditLogs" v-loading="auditLoading" size="small" stripe>
        <el-table-column label="时间" width="135">
          <template #default="{ row }">{{ fmtDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="来源" width="70">
          <template #default="{ row }">
            <el-tag size="small" :type="row.source === 'mcp' ? 'warning' : 'info'">{{ row.source }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="action" label="动作" width="75" />
        <el-table-column prop="entity_name" label="对象" min-width="120" />
        <el-table-column prop="detail" label="说明" min-width="140" />
      </el-table>
    </el-drawer>

    <el-drawer v-model="trashVisible" title="回收站（软删除恢复）" size="min(640px, 94vw)">
      <div v-loading="trashLoading">
        <div v-if="trashEmpty" style="color: #64748b; padding: 20px 0;">回收站暂无数据</div>
        <template v-for="sec in trashSections" :key="sec.type">
          <div v-if="sec.items.length" style="margin-bottom: 16px;">
            <div style="font-weight: 600; margin-bottom: 8px; color: #334155;">{{ sec.label }}</div>
            <el-table :data="sec.items" size="small" border>
              <el-table-column label="名称/摘要" min-width="180">
                <template #default="{ row }">{{ row.name || row.summary || row.follow_type }}</template>
              </el-table-column>
              <el-table-column label="删除时间" width="140">
                <template #default="{ row }">{{ fmtDate(row.deleted_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="85">
                <template #default="{ row }">
                  <el-button type="primary" link size="small" @click="restoreItem(sec.type, row.id)">恢复</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </div>
    </el-drawer>

    <el-dialog v-model="pwdVisible" title="修改密码" width="min(380px, 92vw)">
      <el-form :model="pwdForm" label-position="top">
        <el-form-item label="原密码">
          <el-input v-model="pwdForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码（至少6位）">
          <el-input v-model="pwdForm.new_password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible = false">取消</el-button>
        <el-button type="primary" :loading="pwdLoading" @click="savePwd">保存</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup>
import { computed, reactive, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { ElMessage } from "element-plus"
import { Menu } from "@element-plus/icons-vue"
import { useAuthStore } from "../stores/auth"
import client from "../api/client"
import { fmtDate } from "../utils/constants"

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const mobileNav = ref(false)
const pwdVisible = ref(false)
const pwdLoading = ref(false)
const pwdForm = reactive({ old_password: "", new_password: "" })

const auditVisible = ref(false)
const auditLoading = ref(false)
const auditLogs = ref([])

const trashVisible = ref(false)
const trashLoading = ref(false)
const trashData = reactive({ companies: [], projects: [], contacts: [], follow_ups: [] })

const trashSections = computed(() => [
  { type: "company", label: "客户公司（恢复时连带恢复下级项目与跟进）", items: trashData.companies || [] },
  { type: "project", label: "项目商机", items: trashData.projects || [] },
  { type: "contact", label: "联系人", items: trashData.contacts || [] },
  { type: "follow_up", label: "跟进记录", items: trashData.follow_ups || [] },
])
const trashEmpty = computed(() => trashSections.value.every((s) => s.items.length === 0))

async function openAudit() {
  auditVisible.value = true
  auditLoading.value = true
  try {
    auditLogs.value = await client.get("/audit-logs", { params: { limit: 100 } })
  } finally {
    auditLoading.value = false
  }
}

async function openTrash() {
  trashVisible.value = true
  trashLoading.value = true
  try {
    const res = await client.get("/trash")
    trashData.companies = res.companies || []
    trashData.projects = res.projects || []
    trashData.contacts = res.contacts || []
    trashData.follow_ups = res.follow_ups || []
  } finally {
    trashLoading.value = false
  }
}

async function restoreItem(entityType, entityId) {
  await client.post("/trash/restore", { entity_type: entityType, entity_id: entityId })
  ElMessage.success("已恢复，请刷新列表查看")
  await openTrash()
}

function onCmd(cmd) {
  if (cmd === "audit") openAudit()
  else if (cmd === "trash") openTrash()
  else if (cmd === "pwd") {
    pwdForm.old_password = ""
    pwdForm.new_password = ""
    pwdVisible.value = true
  } else if (cmd === "logout") {
    auth.logout()
    router.push("/login")
  }
}

async function savePwd() {
  if (pwdForm.new_password.length < 6) {
    ElMessage.warning("新密码至少 6 位")
    return
  }
  pwdLoading.value = true
  try {
    await client.post("/auth/change-password", pwdForm)
    ElMessage.success("密码已修改，请重新登录")
    pwdVisible.value = false
    auth.logout()
    router.push("/login")
  } finally {
    pwdLoading.value = false
  }
}
</script>

<style scoped>
.layout-root { min-height: 100vh; }
.top-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #ffffff;
  border-bottom: 1px solid #e2e8f0;
  padding: 0 18px;
  height: 58px;
  position: sticky;
  top: 0;
  z-index: 20;
}
.header-left { display: flex; align-items: center; gap: 16px; }
.menu-trigger { padding: 4px 6px; margin-right: -6px; }
.brand { font-size: 17px; font-weight: 700; color: #1e293b; cursor: pointer; white-space: nowrap; letter-spacing: -0.02em; }
.nav-menu { border-bottom: none !important; height: 57px; }
.header-right { display: flex; align-items: center; gap: 10px; }
.user-btn { cursor: pointer; font-size: 14px; color: #334155; font-weight: 500; padding: 6px 10px; border-radius: 6px; background: #f1f5f9; }
.main-body { padding: 18px; max-width: 1380px; width: 100%; margin: 0 auto; box-sizing: border-box; }
@media (max-width: 767px) {
  .main-body { padding: 12px; }
}
</style>
