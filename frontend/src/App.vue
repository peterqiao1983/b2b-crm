<template>
  <router-view />
</template>

<style>
:root {
  --el-color-primary: #3b5bdb;
  --el-color-primary-light-3: #6b84e6;
  --el-color-primary-light-5: #91a4ee;
  --el-color-primary-light-7: #bac5f5;
  --el-color-primary-light-8: #d1d8f8;
  --el-color-primary-light-9: #eef1fd;
  --el-color-primary-dark-2: #2f49b0;
  --el-border-radius-base: 8px;
  --crm-bg: #f4f6f9;
  --crm-card-shadow: 0 1px 3px rgba(15, 23, 42, 0.05), 0 1px 2px rgba(15, 23, 42, 0.03);
}

html, body, #app {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "PingFang SC", "Microsoft YaHei", sans-serif;
  background-color: var(--crm-bg);
  color: #1e293b;
  -webkit-font-smoothing: antialiased;
}

.el-card {
  border-radius: 12px !important;
  border: 1px solid #e8edf3 !important;
  box-shadow: var(--crm-card-shadow) !important;
}

.num-tabular {
  font-variant-numeric: tabular-nums;
}

.desktop-only {
  display: block;
}
.mobile-only {
  display: none;
}

@media (max-width: 767px) {
  .desktop-only {
    display: none !important;
  }
  .mobile-only {
    display: block !important;
  }
}
</style>
