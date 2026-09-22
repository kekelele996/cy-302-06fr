<template>
  <div class="page-card">
    <div class="page-header">
      <h2>补考申请审核</h2>
      <el-button @click="$router.push('/exams')">返回</el-button>
    </div>

    <div class="toolbar">
      <el-input v-model.number="filterExamId" placeholder="按考试 ID 筛选" clearable style="width: 180px" />
      <el-select v-model="filterStatus" placeholder="审批状态" clearable style="width: 160px">
        <el-option label="待审批" value="pending" />
        <el-option label="已批准" value="approved" />
        <el-option label="已驳回" value="rejected" />
      </el-select>
      <el-button type="primary" @click="load">查询</el-button>
    </div>

    <el-table :data="rows" v-loading="loading" border>
      <el-table-column prop="id" label="申请 ID" width="80" />
      <el-table-column prop="exam_title" label="考试" min-width="160" />
      <el-table-column prop="student_name" label="学生姓名" width="110" />
      <el-table-column prop="student_username" label="用户名" width="120" />
      <el-table-column prop="reason" label="申请理由" min-width="160" show-overflow-tooltip />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTag[row.status]">{{ statusLabel[row.status] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="review_remark" label="审批备注" min-width="120" show-overflow-tooltip />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <template v-if="row.status === 'pending'">
            <el-button type="success" size="small" :loading="actingId === row.id" @click="approve(row)">批准</el-button>
            <el-button type="danger" size="small" @click="reject(row)">驳回</el-button>
          </template>
          <el-button v-else-if="row.status === 'approved' && row.makeup_attempt_id" link type="primary" @click="$router.push(`/grading/${row.exam_id}`)">去批改</el-button>
          <span v-else class="muted">-</span>
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
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { makeupApi } from '../api'
import type { MakeupApplicationItem } from '../types'

const route = useRoute()
const rows = ref<MakeupApplicationItem[]>([])
const total = ref(0)
const loading = ref(false)
const actingId = ref(0)
const filterExamId = ref<number | undefined>(route.query.exam_id ? Number(route.query.exam_id) : undefined)
const filterStatus = ref('pending')
const query = reactive({ page: 1, page_size: 10 })

const statusLabel: Record<string, string> = { pending: '待审批', approved: '已批准', rejected: '已驳回' }
const statusTag: Record<string, string> = { pending: 'warning', approved: 'success', rejected: 'danger' }

async function load() {
  loading.value = true
  try {
    const res = await makeupApi.list({
      page: query.page,
      page_size: query.page_size,
      exam_id: filterExamId.value || undefined,
      status: filterStatus.value || undefined
    })
    rows.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

async function approve(row: MakeupApplicationItem) {
  await ElMessageBox.confirm(`确认批准 ${row.student_name} 的补考申请？批准后将立即生成独立的补考记录。`, '补考审批', { type: 'warning' })
  actingId.value = row.id
  try {
    await makeupApi.approve(row.id)
    ElMessage.success('已批准，补考记录已生成')
    load()
  } finally {
    actingId.value = 0
  }
}

async function reject(row: MakeupApplicationItem) {
  const { value } = await ElMessageBox.prompt('请输入驳回理由（选填）', '驳回补考申请', {
    confirmButtonText: '确认驳回',
    cancelButtonText: '取消',
    inputType: 'textarea'
  })
  await makeupApi.reject(row.id, value || '')
  ElMessage.success('已驳回')
  load()
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}
.pager {
  margin-top: 16px;
  justify-content: flex-end;
}
.muted {
  color: #909399;
}
</style>
