export const COMPANY_STAGES = {
  lead: { label: '早期线索', type: 'info' },
  customer: { label: '正式客户', type: 'primary' },
  key_account: { label: '重点客户', type: 'success' }
}

export const PROJECT_STAGES = {
  initial: { label: '初步沟通', type: 'info' },
  testing: { label: '测试验证', type: 'warning' },
  quoting: { label: '报价谈判', type: 'primary' },
  won: { label: '赢单', type: 'success' },
  lost: { label: '丢单', type: 'danger' }
}

export const FOLLOW_TYPES = {
  visit: '现场拜访',
  call: '电话沟通',
  meeting: '线上/线下会议',
  sample: '送样测试',
  other: '其他'
}

export const ATTITUDES = {
  support: { label: '支持', type: 'success' },
  neutral: { label: '中立', type: 'info' },
  oppose: { label: '反对', type: 'danger' }
}

export function fmtDate(val) {
  if (!val) return '-'
  const d = new Date(val)
  if (Number.isNaN(d.getTime())) return val
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function fmtMoney(val) {
  const n = Number(val || 0)
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 0, maximumFractionDigits: 2 })
}
