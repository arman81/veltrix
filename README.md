# veltrix


<mxfile host="app.diagrams.net">
  <diagram name="Veltrix GPU Node — Vertical Execution Stack">
    <mxGraphModel dx="2600" dy="1600" grid="1" gridSize="10" page="1" pageScale="1" pageWidth="2600" pageHeight="1600">
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>

        <!-- TITLE -->
        <mxCell id="title"
          value="Veltrix GPU Node — Agent-Centric Vertical Execution Stack"
          style="text;html=1;fontSize=28;fontStyle=1;"
          vertex="1" parent="1">
          <mxGeometry x="550" y="20" width="1500" height="60" as="geometry"/>
        </mxCell>

        <!-- NODE BOUNDARY -->
        <mxCell id="nodeBoundary"
          value="GPU Worker Node"
          style="rounded=1;strokeWidth=3;fontSize=22;"
          vertex="1" parent="1">
          <mxGeometry x="80" y="120" width="2440" height="1360" as="geometry"/>
        </mxCell>

        <!-- LAYER 1: KUBERNETES -->
        <mxCell id="layerK8s"
          value="Kubernetes Layer"
          style="rounded=1;strokeWidth=2;fontSize=20;"
          vertex="1" parent="nodeBoundary">
          <mxGeometry x="80" y="80" width="2280" height="200" as="geometry"/>
        </mxCell>

        <mxCell id="pods"
          value="Training Pods | Inference Pods"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerK8s">
          <mxGeometry x="40" y="60" width="900" height="100" as="geometry"/>
        </mxCell>

        <mxCell id="devicePlugin"
          value="GPU Device Plugin"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerK8s">
          <mxGeometry x="1000" y="60" width="500" height="100" as="geometry"/>
        </mxCell>

        <mxCell id="schedulerExt"
          value="K8s Scheduler Extension"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerK8s">
          <mxGeometry x="1560" y="60" width="640" height="100" as="geometry"/>
        </mxCell>

        <!-- LAYER 2: CONTAINER RUNTIME -->
        <mxCell id="layerRuntime"
          value="Container Runtime"
          style="rounded=1;strokeWidth=2;fontSize=20;"
          vertex="1" parent="nodeBoundary">
          <mxGeometry x="80" y="320" width="2280" height="160" as="geometry"/>
        </mxCell>

        <mxCell id="containerd"
          value="containerd / Docker"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerRuntime">
          <mxGeometry x="60" y="40" width="2160" height="80" as="geometry"/>
        </mxCell>

        <!-- LAYER 3: VELTRIX AGENT KERNEL (CENTER) -->
        <mxCell id="layerAgent"
          value="Veltrix Agent Kernel"
          style="rounded=1;strokeWidth=3;fontSize=22;"
          vertex="1" parent="nodeBoundary">
          <mxGeometry x="80" y="520" width="2280" height="260" as="geometry"/>
        </mxCell>

        <mxCell id="agentCore"
          value="Agent Core"
          style="rounded=1;fontSize=20;"
          vertex="1" parent="layerAgent">
          <mxGeometry x="60" y="40" width="500" height="180" as="geometry"/>
        </mxCell>

        <mxCell id="mig"
          value="MIG Controller"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerAgent">
          <mxGeometry x="620" y="40" width="420" height="180" as="geometry"/>
        </mxCell>

        <mxCell id="mps"
          value="CUDA MPS Orchestrator"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerAgent">
          <mxGeometry x="1080" y="40" width="480" height="180" as="geometry"/>
        </mxCell>

        <mxCell id="batch"
          value="Batch Injector"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerAgent">
          <mxGeometry x="1580" y="40" width="320" height="180" as="geometry"/>
        </mxCell>

        <mxCell id="hooks"
          value="CUDA Runtime Hooks"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerAgent">
          <mxGeometry x="1920" y="40" width="300" height="180" as="geometry"/>
        </mxCell>

        <!-- LAYER 4: CUDA -->
        <mxCell id="layerCuda"
          value="CUDA Runtime"
          style="rounded=1;strokeWidth=2;fontSize=20;"
          vertex="1" parent="nodeBoundary">
          <mxGeometry x="80" y="820" width="2280" height="140" as="geometry"/>
        </mxCell>

        <mxCell id="cuda"
          value="CUDA Drivers | Streams | Kernels"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerCuda">
          <mxGeometry x="60" y="30" width="2160" height="80" as="geometry"/>
        </mxCell>

        <!-- LAYER 5: GPU HARDWARE -->
        <mxCell id="layerGpu"
          value="GPU Hardware"
          style="rounded=1;strokeWidth=2;fontSize=20;"
          vertex="1" parent="nodeBoundary">
          <mxGeometry x="80" y="1000" width="2280" height="180" as="geometry"/>
        </mxCell>

        <mxCell id="gpu"
          value="SM | VRAM | MIG Slices | PCIe"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerGpu">
          <mxGeometry x="60" y="40" width="2160" height="100" as="geometry"/>
        </mxCell>

        <!-- LAYER 6: TELEMETRY -->
        <mxCell id="layerTelemetry"
          value="Telemetry Plane"
          style="rounded=1;strokeWidth=2;fontSize=20;"
          vertex="1" parent="nodeBoundary">
          <mxGeometry x="80" y="1220" width="2280" height="200" as="geometry"/>
        </mxCell>

        <mxCell id="nvml"
          value="NVML Metrics"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerTelemetry">
          <mxGeometry x="60" y="60" width="520" height="100" as="geometry"/>
        </mxCell>

        <mxCell id="power"
          value="Power and Thermal"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerTelemetry">
          <mxGeometry x="640" y="60" width="520" height="100" as="geometry"/>
        </mxCell>

        <mxCell id="pcie"
          value="PCIe Throughput"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerTelemetry">
          <mxGeometry x="1220" y="60" width="520" height="100" as="geometry"/>
        </mxCell>

        <mxCell id="otel"
          value="OTEL Exporter"
          style="rounded=1;fontSize=18;"
          vertex="1" parent="layerTelemetry">
          <mxGeometry x="1800" y="60" width="420" height="100" as="geometry"/>
        </mxCell>

        <!-- VERTICAL FLOWS -->
        <mxCell id="f1" style="endArrow=block;strokeWidth=3;" edge="1" parent="1" source="pods" target="containerd">
          <mxGeometry relative="1" as="geometry"/>
        </mxCell>
        <mxCell id="f2" style="endArrow=block;strokeWidth=3;" edge="1" parent="1" source="containerd" target="agentCore">
          <mxGeometry relative="1" as="geometry"/>
        </mxCell>
        <mxCell id="f3" style="endArrow=block;strokeWidth=3;" edge="1" parent="1" source="agentCore" target="cuda">
          <mxGeometry relative="1" as="geometry"/>
        </mxCell>
        <mxCell id="f4" style="endArrow=block;strokeWidth=3;" edge="1" parent="1" source="cuda" target="gpu">
          <mxGeometry relative="1" as="geometry"/>
        </mxCell>
        <mxCell id="f5" style="dashed=1;endArrow=open;strokeWidth=3;" edge="1" parent="1" source="gpu" target="nvml">
          <mxGeometry relative="1" as="geometry"/>
        </mxCell>

      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
