import { useCallback, useRef, useState } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  addEdge,
  useNodesState,
  useEdgesState,
  type Connection,
  type Node,
  type Edge,
  type NodeTypes,
  type OnConnect,
  MarkerType,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import { ShellScriptNode } from './nodes/ShellScriptNode'
import { ShellCommandNode } from './nodes/ShellCommandNode'
import { HttpRequestNode } from './nodes/HttpRequestNode'
import { NodePalette } from './NodePalette'
import { NodeConfigPanel } from './NodeConfigPanel'
import { useWorkflowStore } from '../../store/workflow'
import type { WorkflowSpec } from '../../types'

const nodeTypes: NodeTypes = {
  shellScript: ShellScriptNode,
  shellCommand: ShellCommandNode,
  httpRequest: HttpRequestNode,
}

const defaultNodeData: Record<string, Record<string, unknown>> = {
  shellScript: { label: 'Shell Script', interpreter: 'bash', scriptContent: '' },
  shellCommand: { label: 'Shell Command', command: '' },
  httpRequest: { label: 'HTTP Request', method: 'GET', url: '' },
}

interface WorkflowCanvasProps {
  initialSpec?: WorkflowSpec
  onChange?: (spec: WorkflowSpec) => void
  readOnly?: boolean
}

let nodeIdCounter = 0
function newNodeId() {
  return `node_${++nodeIdCounter}_${Date.now()}`
}

export function WorkflowCanvas({ initialSpec, onChange, readOnly = false }: WorkflowCanvasProps) {
  const reactFlowWrapper = useRef<HTMLDivElement>(null)
  const [reactFlowInstance, setReactFlowInstance] = useState<ReturnType<typeof useNodesState>[2] extends ((...a: any[]) => infer R) ? never : any>(null)
  const [selectedNode, setSelectedNode] = useState<Node | null>(null)

  const store = useWorkflowStore()

  const [nodes, setNodes, onNodesChange] = useNodesState(
    initialSpec?.nodes.map((n) => ({
      id: n.id,
      type: n.type,
      position: n.position,
      data: n.data,
    })) ?? [],
  )
  const [edges, setEdges, onEdgesChange] = useEdgesState(
    initialSpec?.edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      sourceHandle: e.type === 'failure' ? 'failure' : 'success',
      label: e.label,
      markerEnd: { type: MarkerType.ArrowClosed },
      style: e.label === 'failure' ? { stroke: '#f87171' } : { stroke: '#4ade80' },
    })) ?? [],
  )

  const notifyChange = useCallback(
    (ns: Node[], es: Edge[]) => {
      if (onChange) {
        const spec: WorkflowSpec = {
          nodes: ns.map((n) => ({
            id: n.id,
            type: n.type ?? 'shellScript',
            position: n.position,
            data: n.data as Record<string, unknown>,
          })),
          edges: es.map((e) => ({
            id: e.id,
            source: e.source,
            target: e.target,
            type: e.sourceHandle === 'failure' ? 'failure' : 'success',
            label: e.label as string | undefined,
          })),
        }
        onChange(spec)
      }
    },
    [onChange],
  )

  const onConnect: OnConnect = useCallback(
    (params: Connection) => {
      const label = params.sourceHandle === 'failure' ? 'failure' : 'success'
      const newEdge: Edge = {
        ...params,
        id: `edge_${params.source}_${params.sourceHandle}_${params.target}`,
        label,
        markerEnd: { type: MarkerType.ArrowClosed },
        style: label === 'failure' ? { stroke: '#f87171' } : { stroke: '#4ade80' },
      } as Edge
      setEdges((eds) => {
        const updated = addEdge(newEdge, eds)
        notifyChange(nodes, updated)
        return updated
      })
    },
    [nodes, setEdges, notifyChange],
  )

  // Drag-and-drop from NodePalette
  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault()
      const type = event.dataTransfer.getData('application/reactflow')
      if (!type || !reactFlowInstance || !reactFlowWrapper.current) return

      const rect = reactFlowWrapper.current.getBoundingClientRect()
      const position = reactFlowInstance.screenToFlowPosition({
        x: event.clientX - rect.left,
        y: event.clientY - rect.top,
      })

      const newNode: Node = {
        id: newNodeId(),
        type,
        position,
        data: { ...(defaultNodeData[type] ?? { label: type }) },
      }

      setNodes((nds) => {
        const updated = [...nds, newNode]
        notifyChange(updated, edges)
        return updated
      })
    },
    [reactFlowInstance, edges, setNodes, notifyChange],
  )

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    if (!readOnly) setSelectedNode(node)
  }, [readOnly])

  const onPaneClick = useCallback(() => {
    setSelectedNode(null)
  }, [])

  // Update node data from config panel
  store.updateNodeData = (nodeId: string, data: Record<string, unknown>) => {
    setNodes((nds) =>
      nds.map((n) => (n.id === nodeId ? { ...n, data: { ...n.data, ...data } } : n)),
    )
    setSelectedNode(null)
  }

  return (
    <div className="flex h-full w-full overflow-hidden">
      {/* Left: Node Palette */}
      {!readOnly && <NodePalette />}

      {/* Center: React Flow canvas */}
      <div ref={reactFlowWrapper} className="flex-1 h-full" onDragOver={onDragOver} onDrop={onDrop}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={(changes) => {
            onNodesChange(changes)
          }}
          onEdgesChange={(changes) => {
            onEdgesChange(changes)
          }}
          onConnect={onConnect}
          onInit={setReactFlowInstance}
          onNodeClick={onNodeClick}
          onPaneClick={onPaneClick}
          nodeTypes={nodeTypes}
          fitView
          deleteKeyCode={readOnly ? null : 'Backspace'}
          nodesDraggable={!readOnly}
          nodesConnectable={!readOnly}
          elementsSelectable={!readOnly}
        >
          <Background gap={16} size={1} color="#e2e8f0" />
          <Controls />
          <MiniMap
            nodeStrokeWidth={3}
            zoomable
            pannable
            className="!bg-slate-50 !border-slate-200"
          />
        </ReactFlow>
      </div>

      {/* Right: Node Config Panel */}
      {!readOnly && selectedNode && (
        <NodeConfigPanel node={selectedNode} onClose={() => setSelectedNode(null)} />
      )}
    </div>
  )
}
