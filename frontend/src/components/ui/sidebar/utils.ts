import type { ComputedRef, Ref } from 'vue'
import { createContext } from 'reka-ui'

export const SIDEBAR_WIDTH = '14rem'
export const SIDEBAR_WIDTH_ICON = '3rem'

export const [useSidebar, provideSidebarContext] = createContext<{
  state: ComputedRef<'expanded' | 'collapsed'>
  open: Ref<boolean>
  setOpen: (open: boolean) => void
  toggleSidebar: () => void
}>('Sidebar')
