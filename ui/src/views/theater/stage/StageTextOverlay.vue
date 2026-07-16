<script setup lang="ts">
import { computed } from 'vue'
import type { CameraState, StageObject } from '../shared/stage-types'
import StageTextVisualObject from './StageTextVisualObject.vue'

const props = defineProps<{
  objects: Record<string, StageObject>
  camera: CameraState
  viewportWidth: number
  viewportHeight: number
}>()

const roots = computed(() => Object.values(props.objects)
  .filter((object) => object.parentId === null && object.visible)
  .sort((a, b) => a.transform.z - b.transform.z || a.transform.order - b.transform.order))

const cameraStyle = computed(() => ({
  transform: `translate(${props.viewportWidth / 2 + props.camera.x}px, ${props.viewportHeight / 2 + props.camera.y}px) scale(${props.camera.zoom})`,
}))
</script>

<template>
  <div class="theater-text-overlay">
    <div class="theater-text-overlay__camera" :style="cameraStyle">
      <StageTextVisualObject
        v-for="object in roots"
        :key="object.id"
        :object="object"
        :objects="props.objects"
      />
    </div>
  </div>
</template>

<style scoped>
.theater-text-overlay {
  position: absolute;
  z-index: 2;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.theater-text-overlay__camera {
  position: absolute;
  top: 0;
  left: 0;
  width: 0;
  height: 0;
  transform-origin: 0 0;
}
</style>
