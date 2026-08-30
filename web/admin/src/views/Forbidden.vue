<template>
  <el-card class="forbidden-card">
    <el-result icon="warning" title="403" sub-title="您没有访问该页面的权限">
      <template #extra>
        <el-button type="primary" @click="$router.push('/dashboard')">返回首页</el-button>
      </template>
    </el-result>
  </el-card>
</template>

<style scoped>
.forbidden-card {
  max-width: 640px;
  margin: 60px auto;
}
</style>
