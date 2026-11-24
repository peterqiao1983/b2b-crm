import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '../views/LoginView.vue'
import LayoutView from '../views/LayoutView.vue'
import DashboardView from '../views/DashboardView.vue'
import CalendarView from '../views/CalendarView.vue'
import CompaniesView from '../views/CompaniesView.vue'
import ProjectsView from '../views/ProjectsView.vue'
import ContactsView from '../views/ContactsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: LoginView },
    {
      path: '/',
      component: LayoutView,
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', component: DashboardView },
        { path: 'calendar', component: CalendarView },
        { path: 'companies', component: CompaniesView },
        { path: 'projects', component: ProjectsView },
        { path: 'contacts', component: ContactsView }
      ]
    }
  ]
})

router.beforeEach((to) => {
  const token = localStorage.getItem('crm_token')
  if (to.path !== '/login' && !token) {
    return '/login'
  }
})

export default router
