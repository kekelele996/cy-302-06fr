<template>
  <div class="page-card">
    <div class="page-header">
      <h2>成绩报告</h2>
      <el-button @click="$router.push('/attempts')">返回考试记录</el-button>
    </div>

    <template v-if="report">
      <el-alert
        v-if="report.kind === 'makeup'"
        title="这是补考记录"
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 12px"
      />
      <el-alert
        :title="effectiveBanner.title"
        :type="report.is_effective ? 'success' : 'info'"
        :closable="false"
        show-icon
        style="margin-bottom: 12px"
      />
      <el-descriptions :column="3" border>
        <el-descriptions-item label="考试">{{ report.exam_title }}</el-descriptions-item>
        <el-descriptions-item label="记录类型">{{ report.kind === 'makeup' ? '补考' : '正考' }}</el-descriptions-item>
        <el-descriptions-item label="本次卷面总分">{{ report.total_score }}</el-descriptions-item>
        <el-descriptions-item label="有效成绩（两次最高）">{{ report.effective_score }}</el-descriptions-item>
        <el-descriptions-item label="客观题得分">{{ report.objective_score }}</el-descriptions-item>
        <el-descriptions-item label="主观题得分">{{ report.subjective_score }}</el-descriptions-item>
        <el-descriptions-item label="客观题正确率">{{ report.accuracy }}%</el-descriptions-item>
        <el-descriptions-item label="排名">{{ report.rank }} / {{ report.participants }}</el-descriptions-item>
        <el-descriptions-item label="有效成绩来源">{{ report.effective_kind === 'makeup' ? '补考' : '正考' }}</el-descriptions-item>
      </el-descriptions>

      <el-row :gutter="16" style="margin-top: 16px">
        <el-col :span="12">
          <div ref="chartRef" style="height: 320px"></div>
        </el-col>
        <el-col :span="12">
          <el-table :data="report.type_breakdown" border>
            <el-table-column prop="name" label="题型" />
            <el-table-column label="得分" width="120">
              <template #default="{ row }">{{ row.score }} / {{ row.max }}</template>
            </el-table-column>
            <el-table-column prop="count" label="题数" width="80" />
          </el-table>
        </el-col>
      </el-row>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import * as echarts from 'echarts'
import { attemptApi } from '../api'
import type { ReportResponse } from '../types'

const route = useRoute()
const report = ref<ReportResponse | null>(null)
const chartRef = ref<HTMLElement | null>(null)

const effectiveBanner = computed(() => {
  if (!report.value) return { title: '' }
  if (report.value.is_effective) {
    return { title: `本次成绩为有效成绩：${report.value.effective_score} 分（正考/补考中的最高总分）` }
  }
  return { title: `本次卷面 ${report.value.total_score} 分，有效成绩取另一次的 ${report.value.effective_score} 分；原始记录仍可追溯` }
})

onMounted(async () => {
  const attemptId = Number(route.params.attemptId)
  report.value = await attemptApi.report(attemptId)
  await nextTick()
  renderChart()
})

function renderChart() {
  if (!chartRef.value || !report.value) return
  const chart = echarts.init(chartRef.value)
  const data = report.value.type_breakdown.map((item) => ({
    name: item.name,
    value: item.score
  }))
  chart.setOption({
    title: { text: '各题型得分分布', left: 'center' },
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        type: 'pie',
        radius: ['35%', '65%'],
        data,
        label: { formatter: '{b}: {c}分' }
      }
    ]
  })
}
</script>
