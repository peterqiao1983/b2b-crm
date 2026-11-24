<template>
  <el-container class="layout-root">
    <el-header class="top-header">
      <div class="header-left">
        <span class="brand" @click="router.push('/dashboard')">B2B CRM</span>
        <el-menu mode="horizontal" :default-active="route.path" :ellipsis="false" router class="nav-menu">
          <el-menu-item index="/dashboard">工作台</el-menu-item>
          <el-menu-item index="/calendar">日历排期</el-menu-item>
          <el-menu-item index="/companies">客户公司</el-menu-item>
          <el-menu-item index="/projects">项目商机</el-menu-item>
          <el-menu-item index="/contacts">联系人</el-menu-item>
        </el-menu>
      </div>
      <div class="header-right">
        <el-dropdown @command="onCmd">
          <span class="user-btn">
            {{ auth.user?.display_name || auth.user?.username || '用户' }}
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="pwd">修改密码</el-dropdown-item>
              <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>

    <el-main class="main-body">
      <router-view />
    </el-main>

    <el-dialog v-model="pwdVisible" title="修改密码" width="380px">
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
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import client from '../api/client'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const pwdVisible = ref(false)
const pwdLoading = ref(false)
const pwdForm = reactive({ old_password: '', new_password: '' })

function onCmd(cmd) {
  if (cmd === 'logout') {
    auth.logout()
    router.push('/login')
  } else if (cmd === 'pwd') {
    pwdForm.old_password = ''
    pwdForm.new_password = ''
    pwdVisible.value = true
  }
}

async function savePwd() {
  pwdLoading.value = true
  try {
    await client.post('/auth/change-password', pwdForm)
    ElMessage.success('密码已更新')
    pwdVisible.value = false
  } finally {
    pwdLoading.value = false
  }
}
</script>

<style scoped>
.layout-root {
  min-height: 100vh;
}
.top-header {
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 56px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  overflow-x: auto;
}
.brand {
  font-weight: 700;
  font-size: 16px;
  color: #2563eb;
  cursor: pointer;
  white-space: nowrap;
}
.nav-menu {
  border-bottom: none !important;
  height: 54px;
}
.header-right {
  flex-shrink: 0;
}
.user-btn {
  cursor: pointer;
  font-size: 14px;
  color: #374151;
  padding: 6px 10px;
}
.main-body {
  max-width: 1320px;
  width: 100%;
  margin: 0 auto;
  padding: 16px;
}
</style>
