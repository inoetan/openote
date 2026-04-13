import { create } from 'zustand'
import type { Node, Edge } from '@xyflow/react'
import type { WorkflowSpec } from '@/types'

interface WorkflowState {
  nodes: Node[]
  edges: Edge[]
  selectedNodeId: string | null
  isDirty: boolean

  setNodes: (nodes: Node[]) => void
  setEdges: (edges: Edge[]) => void
  selectNode: (id: string | null) => void
  loadWorkflow: (spec: WorkflowSpec) => void
  toWorkflowSpec: () => WorkflowSpec
  markClean: () => void
  /** Updated imperatively from WorkflowCanvas after config panel save */
  updateNodeData: (nodeId: string, data: Record<string, unknown>) => void
}

export const useWorkflowStore = create<WorkflowState>((set, get) => ({
  nodes: [],
  edges: [],
  selectedNodeId: null,
  isDirty: false,

  setNodes: (nodes) => set({ nodes, isDirty: true }),

  setEdges: (edges) => set({ edges, isDirty: true }),

  selectNode: (id) => set({ selectedNodeId: id }),

  loadWorkflow: (spec: WorkflowSpec) => {
    const nodes: Node[] = spec.nodes.map((n) => ({
      id: n.id,
      type: n.type,
      position: n.position,
      data: n.data,
    }))
    const edges: Edge[] = spec.edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      type: e.type,
      label: e.label,
    }))
    set({ nodes, edges, isDirty: false, selectedNodeId: null })
  },

  toWorkflowSpec: (): WorkflowSpec => {
    const { nodes, edges } = get()
    return {
      nodes: nodes.map((n) => ({
        id: n.id,
        type: n.type ?? 'shellCommand',
        position: n.position,
        data: n.data,
      })),
      edges: edges.map((e) => ({
        id: e.id,
        source: e.source,
        target: e.target,
        type: e.type,
        label: typeof e.label === 'string' ? e.label : undefined,
      })),
    }
  },

  markClean: () => set({ isDirty: false }),

  // Default no-op; WorkflowCanvas overrides this at runtime via store reference
  updateNodeData: (_nodeId, _data) => {},
}))
