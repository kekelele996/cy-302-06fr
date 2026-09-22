<template>
  <div class="page-card">
    <div class="page-header">
      <h2>考试管理</h2>
      <el-button v-if="isStaff" type="primary" @click="openCreate">创建考试</el-button>
    </div>

    <el-table :data="rows" v-loading="loading" border>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="title" label="考试名称" min-width="180" />
      <el-table-column prop="duration_minutes" label="时长(分钟)" width="100" />
      <el-table-column prop="total_score" label="总分" width="80" />
      <el-table-column prop="question_count" label="题数" width="80" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTag[row.status as keyof typeof statusTag]">{{ statusLabels[row.status as keyof typeof statusLabels] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" min-width="300" fixed="right">
        <template #default="{ row }">
          <template v-if="isStudent">
            <el-button v-if="row.status === 'published'" type="primary" size="small" @click="$router.push(`/exam/${row.id}/take`)">开始考试</el-button>
            <el-button v-if="row.status === 'closed'" size="small" @click="openMakeup(row)">补考</el-button>
          </template>
          <template v-else>
            <el-button size="small" @click="viewQuestions(row)">题目</el-button>
            <el-button v-if="row.status === 'draft'" type="success" size="small" @click="publish(row)">发布</el-button>
            <el-button v-if="row.status === 'published'" type="warning" size="small" @click="closeExam(row)">关闭</el-button>
            <el-button size="small" @click="viewStats(row)">统计</el-button>
            <el-button size="small" type="info" @click="$router.push(`/grading/${row.id}`)">批改</el-button>
            <el-button size="small" type="primary" plain @click="$router.push(`/makeup-review?exam_id=${row.id}`)">补考审核</el-button>
            <el-button size="small" type="danger" @click="removeExam(row)">删除</el-button>
          </template>
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

    <el-dialog v-model="dialogVisible" title="创建考试（自动组卷）" width="760px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="考试名称">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="考试时长">
          <el-input-number v-model="form.duration_minutes" :min="1" />
          <span style="margin-left: 8px">分钟</span>
        </el-form-item>
        <el-form-item label="总分">
          <el-input-number v-model="form.total_score" :min="0" :step="1" />
          <span style="margin-left: 8px">（0 表示自动计算）</span>
        </el-form-item>
        <el-form-item label="组卷参数">
          <div style="width: 100%">
            <div v-for="(cfg, idx) in form.question_config" :key="idx" class="config-row">
              <el-select v-model="cfg.type" style="width: 120px">
                <el-option v-for="(label, key) in typeLabels" :key="key" :label="label" :value="key" />
              </el-select>
              <el-input-number v-model="cfg.count" :min="1" placeholder="数量" />
              <el-input-number v-model="cfg.score" :min="0.5" :step="0.5" placeholder="分值" />
              <el-select v-model="cfg.difficulty" placeholder="难度" clearable style="width: 110px">
                <el-option v-for="(label, key) in difficultyLabels" :key="key" :label="label" :value="key" />
              </el-select>
              <el-button link type="danger" @click="form.question_config.splice(idx, 1)">删除</el-button>
            </div>
            <el-button size="small" @click="addConfig">添加题型</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="questionsVisible" title="试卷题目" width="800px">
      <el-table :data="questions" border max-height="500">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column label="题型" width="90">
          <template #default="{ row }">{{ typeLabels[row.question.type as keyof typeof typeLabels] }}</template>
        </el-table-column>
        <el-table-column prop="question.content" label="题干" min-width="220" show-overflow-tooltip />
        <el-table-column prop="score" label="分值" width="70" />
        <el-table-column prop="question.analysis" label="解析" min-width="120" show-overflow-tooltip />
      </el-table>
    </el-dialog>

    <el-dialog v-model="statsVisible" title="成绩统计" width="900px">
      <div v-if="stats">
        <el-tabs>
          <el-tab-pane label="有效成绩（按两次最高）">
            <el-descriptions :column="3" border>
              <el-descriptions-item label="考试">{{ stats.exam_title }}</el-descriptions-item>
              <el-descriptions-item label="有效参与人数">{{ stats.effective.count }}</el-descriptions-item>
              <el-descriptions-item label="缺考人数">{{ stats.absent_count }}</el-descriptions-item>
              <el-descriptions-item label="平均分">{{ stats.effective.average_score }}</el-descriptions-item>
              <el-descriptions-item label="最高分">{{ stats.effective.highest_score }}</el-descriptions-item>
              <el-descriptions-item label="最低分">{{ stats.effective.lowest_score }}</el-descriptions-item>
              <el-descriptions-item label="及格人数（≥60%）">{{ stats.effective.pass_count }}</el-descriptions-item>
            </el-descriptions>
            <el-table :data="stats.ranking" border style="margin-top: 12px" max-height="300">
              <el-table-column prop="rank" label="排名" width="70" />
              <el-table-column prop="student_name" label="姓名" />
              <el-table-column prop="student_username" label="用户名" />
              <el-table-column label="类型" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.kind === 'makeup' ? 'warning' : 'info'" size="small">{{ row.kind === 'makeup' ? '补考' : '正考' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="total_score" label="有效总分" width="100" />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="原始记录">
            <el-descriptions :column="3" border>
              <el-descriptions-item label="原始交卷份数">{{ stats.original.count }}</el-descriptions-item>
              <el-descriptions-item label="原始平均分">{{ stats.original.average_score }}</el-descriptions-item>
              <el-descriptions-item label="原始及格份数">{{ stats.original.pass_count }}</el-descriptions-item>
            </el-descriptions>
            <el-table :data="stats.raw_attempts" border style="margin-top: 12px" max-height="300">
              <el-table-column prop="attempt_id" label="记录ID" width="80" />
              <el-table-column prop="student_name" label="姓名" />
              <el-table-column prop="student_username" label="用户名" />
              <el-table-column label="类型" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.kind === 'makeup' ? 'warning' : 'info'" size="small">{{ row.kind === 'makeup' ? '补考' : '正考' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="90">
                <template #default="{ row }">
                  <el-tag :type="rawStatusTag[row.status]" size="small">{{ rawStatusLabel[row.status] }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="total_score" label="卷面总分" width="90" />
              <el-table-column label="是否有效" width="100">
                <template #default="{ row }">
                  <el-tag v-if="row.is_effective" type="success" size="small">有效</el-tag>
                  <span v-else class="muted">否</span>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>

    <el-dialog v-model="makeupVisible" title="补考申请" width="520px">
      <div v-loading="makeupLoading">
        <el-alert v-if="makeupInfo" :title="makeupInfo.reason" :type="makeupInfo.eligible ? 'success' : 'info'" :closable="false" show-icon style="margin-bottom: 12px" />
        <template v-if="makeupInfo?.application">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="审批状态">
              <el-tag :type="makeupStatusTag[makeupInfo.application.status]">{{ makeupStatusLabel[makeupInfo.application.status] }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="申请理由">{{ makeupInfo.application.reason || '-' }}</el-descriptions-item>
            <el-descriptions-item v-if="makeupInfo.application.review_remark" label="审批备注">{{ makeupInfo.application.review_remark }}</el-descriptions-item>
          </el-descriptions>
          <div style="margin-top: 14px; text-align: right">
            <el-button
              v-if="makeupInfo.application.status === 'approved'"
              type="primary"
              @click="startMakeup"
            >进入 / 继续补考</el-button>
          </div>
        </template>
        <template v-else>
          <el-input v-model="makeupReason" type="textarea" :rows="3" placeholder="请填写补考理由（选填）" />
          <div style="margin-top: 14px; text-align: right">
            <el-button type="primary" :disabled="!makeupInfo?.eligible" :loading="makeupSaving" @click="submitMakeup">提交申请</el-button>
          </div>
        </template>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { examApi, makeupApi } from '../api'
import { useAuthStore } from '../stores/auth'
import type { Exam, PaperQuestionConfig, ExamStatResponse, MakeupEligibility } from '../types'

const router = useRouter()
const typeLabels: Record<string, string> = {
  single: '单选题',
  multiple: '多选题',
  true_false: '判断题',
  fill_blank: '填空题',
  short_answer: '简答题'
}
const difficultyLabels: Record<string, string> = { easy: '简单', medium: '中等', hard: '困难' }
const statusLabels: Record<string, string> = { draft: '草稿', published: '已发布', closed: '已关闭' }
const statusTag: Record<string, string> = { draft: 'info', published: 'success', closed: 'warning' }
const rawStatusLabel: Record<string, string> = { in_progress: '进行中', submitted: '已交卷', absent: '缺考' }
const rawStatusTag: Record<string, string> = { in_progress: 'warning', submitted: 'success', absent: 'danger' }
const makeupStatusLabel: Record<string, string> = { pending: '待审批', approved: '已批准', rejected: '已驳回' }
const makeupStatusTag: Record<string, string> = { pending: 'warning', approved: 'success', rejected: 'danger' }

const auth = useAuthStore()
const isStudent = computed(() => auth.role === 'student')
const isStaff = computed(() => auth.role === 'admin' || auth.role === 'teacher')

const loading = ref(false)
const saving = ref(false)
const rows = ref<Exam[]>([])
const total = ref(0)
const dialogVisible = ref(false)
const questionsVisible = ref(false)
const statsVisible = ref(false)
const questions = ref<{ id: number; score: number; question: { type: string; content: string; analysis?: string } }[]>([])
const stats = ref<ExamStatResponse | null>(null)
const query = reactive({ page: 1, page_size: 10, status: '', keyword: '' })

const makeupVisible = ref(false)
const makeupLoading = ref(false)
const makeupSaving = ref(false)
const makeupInfo = ref<MakeupEligibility | null>(null)
const makeupReason = ref('')
const makeupExamId = ref(0)

const form = reactive({
  title: '',
  description: '',
  duration_minutes: 60,
  total_score: 100,
  question_config: [] as PaperQuestionConfig[]
})

function addConfig() {
  form.question_config.push({ type: 'single', count: 5, score: 2, difficulty: 'easy' })
}

function openCreate() {
  form.title = ''
  form.description = ''
  form.duration_minutes = 60
  form.total_score = 100
  form.question_config = [
    { type: 'single', count: 5, score: 2, difficulty: 'easy' },
    { type: 'true_false', count: 5, score: 1, difficulty: 'easy' }
  ]
  dialogVisible.value = true
}

async function onSave() {
  saving.value = true
  try {
    await examApi.create({
      title: form.title,
      description: form.description,
      duration_minutes: form.duration_minutes,
      total_score: form.total_score,
      question_config: form.question_config
    })
    ElMessage.success('创建成功')
    dialogVisible.value = false
    load()
  } finally {
    saving.value = false
  }
}

async function publish(row: Exam) {
  await examApi.publish(row.id)
  ElMessage.success('发布成功')
  load()
}

async function closeExam(row: Exam) {
  await examApi.close(row.id)
  ElMessage.success('已关闭')
  load()
}

async function removeExam(row: Exam) {
  await ElMessageBox.confirm('确认删除该考试？', '提示', { type: 'warning' })
  await examApi.remove(row.id)
  ElMessage.success('删除成功')
  load()
}

async function viewQuestions(row: Exam) {
  questions.value = await examApi.questions(row.id)
  questionsVisible.value = true
}

async function viewStats(row: Exam) {
  stats.value = await examApi.stats(row.id)
  statsVisible.value = true
}

async function openMakeup(row: Exam) {
  makeupExamId.value = row.id
  makeupReason.value = ''
  makeupInfo.value = null
  makeupVisible.value = true
  makeupLoading.value = true
  try {
    makeupInfo.value = await makeupApi.eligibility(row.id)
  } finally {
    makeupLoading.value = false
  }
}

async function submitMakeup() {
  makeupSaving.value = true
  try {
    await makeupApi.apply(makeupExamId.value, makeupReason.value)
    ElMessage.success('补考申请已提交，等待教师审批')
    makeupInfo.value = await makeupApi.eligibility(makeupExamId.value)
  } finally {
    makeupSaving.value = false
  }
}

function startMakeup() {
  makeupVisible.value = false
  router.push(`/exam/${makeupExamId.value}/take?makeup=1`)
}

async function load() {
  loading.value = true
  try {
    const res = await examApi.list({ ...query })
    rows.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.config-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.pager {
  margin-top: 16px;
  justify-content: flex-end;
}
.muted {
  color: #909399;
  font-size: 12px;
}
</style>
