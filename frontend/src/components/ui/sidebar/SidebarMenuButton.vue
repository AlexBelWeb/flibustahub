<script setup lang="ts">
import type { PrimitiveProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { Primitive } from 'reka-ui'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { useSidebar } from './utils'

const props = withDefaults(
  defineProps<
    PrimitiveProps & {
      class?: HTMLAttributes['class']
      isActive?: boolean
      tooltip?: string
    }
  >(),
  { as: 'button' },
)

const { state } = useSidebar()
</script>

<template>
  <Tooltip :delay-duration="0" :disabled="state === 'expanded' || !tooltip">
    <TooltipTrigger as-child>
      <Primitive
        :as="as"
        :as-child="asChild"
        data-sidebar="menu-button"
        :data-active="isActive"
        :class="
          cn(
            'flex w-full items-center gap-2 overflow-hidden rounded-lg p-2 text-left text-sm outline-none ring-sidebar-ring hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 [&>svg]:size-4 [&>svg]:shrink-0',
            'group-data-[collapsible=icon]:size-8 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:p-2 group-data-[collapsible=icon]:[&>span]:hidden',
            isActive
              ? 'bg-sidebar-accent font-medium text-sidebar-accent-foreground'
              : 'text-muted-foreground',
            props.class,
          )
        "
      >
        <slot />
      </Primitive>
    </TooltipTrigger>
    <TooltipContent v-if="tooltip" side="right" :hidden="state !== 'collapsed'">
      {{ tooltip }}
    </TooltipContent>
  </Tooltip>
</template>
