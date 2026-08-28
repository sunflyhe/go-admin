<template>
  <el-result icon="warning" title="403" sub-title="您没有访问该页面的权限">
    <template #extra>
      <el-button type="primary" @click="$router.push('/dashboard')">返回首页</el-button>
    </template>
  </el-result>
</template>
