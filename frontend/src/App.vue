<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppToaster from '@/components/AppToaster.vue'
</script>

<template>
  <div class="min-h-screen bg-background text-foreground">
    <RouterView />
    <AppToaster />
  </div>
</template>
