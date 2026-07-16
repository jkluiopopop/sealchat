<script setup lang="ts">
import { computed } from 'vue'
import RichTextContent from '@/components/rich-text/RichTextContent.vue'
import { WORLD_UNIT_PX, type StageObject } from '../shared/stage-types'

defineOptions({ name: 'StageTextVisualObject' })

const props = defineProps<{
  object: StageObject
  objects: Record<string, StageObject>
}>()

const children = computed(() => Object.values(props.objects)
  .filter((object) => object.parentId === props.object.id && object.visible)
  .sort((a, b) => a.transform.z - b.transform.z || a.transform.order - b.transform.order))

const style = computed(() => {
  const transform = props.object.transform
  return {
    left: `${transform.x * WORLD_UNIT_PX}px`,
    top: `${transform.y * WORLD_UNIT_PX}px`,
    width: `${Math.max(0.5, transform.width) * WORLD_UNIT_PX}px`,
    height: `${Math.max(0.5, transform.height) * WORLD_UNIT_PX}px`,
    transform: `translate(-50%, -50%) rotate(${transform.rotation}deg) scale(${transform.scaleX}, ${transform.scaleY})`,
  }
})
</script>

<template>
  <div class="theater-text-visual-object" :style="style">
    <RichTextContent
      v-if="props.object.type === 'text'"
      class="theater-text-visual-object__content"
      :content="props.object.text || props.object.name"
      autoplay
    />
    <StageTextVisualObject
      v-for="child in children"
      :key="child.id"
      :object="child"
      :objects="props.objects"
    />
  </div>
</template>

<style scoped>
.theater-text-visual-object {
  position: absolute;
  transform-origin: center;
  pointer-events: none;
}

.theater-text-visual-object__content {
  width: 100%;
  height: 100%;
  padding: 10px;
  overflow: hidden;
  color: #fff;
  font-size: 28px;
  font-weight: 700;
  line-height: 1.3;
}

.theater-text-visual-object__content :deep(a),
.theater-text-visual-object__content :deep(.tiptap-spoiler),
.theater-text-visual-object__content :deep(.tiptap-ruby[data-ruby-spoiler='true']) {
  pointer-events: auto;
}
</style>
