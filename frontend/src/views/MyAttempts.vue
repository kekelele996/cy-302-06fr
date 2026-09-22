<template>
  <div class="page-card">
    <div class="page-header">
      <h2>考试记录</h2>
    </div>
    <el-table :data="rows" v-loading="loading" border>
      <el-table-column prop="attempt_id" label="记录 ID" width="90" />
      <el-table-column prop="exam_title" label="考试名称" min-width="160" />
      <el-table-column label="类型" width="90">
        <template #default="{ row }">
          <el-tag v-if="row.kind === 'makeup'" type="warning" size="small">补考</el-tag>
          <el-tag v-else type="info" size="small">正考</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTag[row.status]">{{ statusLabel[row.status] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="卷面总分" width="90">
        <template #default="{ row }">{{ row.status === 'absent' ? '-' : row.total_score }}</template>
      </el-table-column>
      <el-table-column label="有效成绩" width="120">
        <template #default="{ row }">
          <el-tag v-if="row.is_effective" type="success" size="small">有效 {{ row.effective_score }}</el-tag>
          <span v-else-if="row.status === 'submitted'" class="muted">{{ row.effective_score }}（取另一次）</span>
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>
      <el-table-column label="交卷时间" min-width="170">
        <template #default="{ row }">{{ formatTime(row.submitted_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button v-if="row.status === 'submitted'" link type="primary" @click="$router.push(`/report/${row.attempt_id}`)">成绩报告</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination
      class="pager"
      v-model:current-page="query.page"
      v-model:page-size="query.page_size"
      :total="total"
      layout="total, prev, pager, next"
      @current-change="load"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { attemptApi } from '../api'
import type { AttemptSummary } from '../types'

const rows = ref<AttemptSummary[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ page: 1, page_size: 10 })

const statusLabel: Record<string, string> = { in_progress: '进行中', submitted: '已交卷', absent: '缺考' }
const statusTag: Record<string, string> = { in_progress: 'warning', submitted: 'success', absent: 'danger' }

function formatTime(v?: string | null) {
  return v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-'
}

async function load() {
  loading.value = true
  try {
    const res = await attemptApi.list({ ...query })
    rows.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.pager {
  margin-top: 16px;
  justify-content: flex-end;
}
.muted {
  color: #909399;
  font-size: 12px;
}
</style>
