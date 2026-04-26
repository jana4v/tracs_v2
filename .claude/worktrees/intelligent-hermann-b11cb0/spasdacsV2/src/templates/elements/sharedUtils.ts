import * as go from "gojs";

const $ = go.GraphObject.make;

export function selectionAdornment() {
  return $(
    go.Adornment, "Auto",
    $(go.Shape, { fill: null, stroke: "#2196F3", strokeWidth: 2, strokeDashArray: [4, 2] }),
    $(go.Placeholder)
  );
}

/**
 * Show or hide the 4 directional port spots and highlight named input/output
 * port connectors. Called from node mouseEnter / mouseLeave handlers.
 */
export function showSmallPorts(node: go.Node, show: boolean): void {
  // Show/hide the 4 directional port spots (T, B, L, R)
  for (const id of ["T", "B", "L", "R"]) {
    const spot = node.findObject("PORT_" + id) as go.Shape | null;
    if (spot) spot.opacity = show ? 1 : 0;
  }
  // Highlight named input/output port connector squares
  node.ports.each((port: go.GraphObject) => {
    if (port.portId !== "" && !["T","B","L","R"].includes(port.portId)) {
      const ps = (port as go.Panel).findObject("PS") as go.Shape;
      if (ps) {
        ps.fill   = show ? "rgba(52,152,219,0.35)" : "white";
        ps.stroke = show ? "#2980b9" : "#555";
      }
    }
  });
}

export function nameLabel(opts: Partial<go.TextBlock> = {}) {
  const tb = $(
    go.TextBlock,
    {
      alignment: go.Spot.Bottom,
      alignmentFocus: new go.Spot(0.5, 0, 0, 4),
      font: "11px sans-serif",
      stroke: "#333333",
      background: "rgba(255,255,255,0.75)",
      cursor: "move",
      editable: true,
      ...opts,
    },
    new go.Binding("text", "name").makeTwoWay()
  );
  // Required by NodeLabelDraggingTool — marks this as a draggable node label
  (tb as any)._isNodeLabel = true;
  return tb;
}

/**
 * Input port — lives INSIDE the body, left edge.
 * Layout: [connector □][label text →]
 * The connector (□) is at the left edge of the body; the label extends inward.
 * Links arrive from the left and terminate at the connector.
 */
export function makeInputPort(portName: string): go.Panel {
  return $(go.Panel, "Horizontal",
    {
      portId: portName,
      name: portName,
      toLinkable: true,
      fromLinkable: false,
      toSpot: go.Spot.Left,   // link arrives at left edge of this port panel
      cursor: "crosshair",
      margin: new go.Margin(3, 0),
      mouseEnter: (_e: go.InputEvent, obj: go.GraphObject) => {
        const s = (obj as go.Panel).findObject("PS") as go.Shape;
        if (s) { s.fill = "rgba(52,152,219,0.65)"; s.stroke = "#2980b9"; }
      },
      mouseLeave: (_e: go.InputEvent, obj: go.GraphObject) => {
        const s = (obj as go.Panel).findObject("PS") as go.Shape;
        if (s) { s.fill = "white"; s.stroke = "#555"; }
      },
    },
    // connector rectangle sits flush at the body's left border
    $(go.Shape, "Rectangle", {
      name: "PS",
      desiredSize: new go.Size(8, 14),
      fill: "white", stroke: "#555", strokeWidth: 1,
    }),
    // label is inside the body
    $(go.TextBlock, portName, {
      font: "bold 10px sans-serif",
      stroke: "#eee",
      margin: new go.Margin(0, 3, 0, 2),
    })
  );
}

/**
 * Output port — lives INSIDE the body, right edge.
 * Layout: [← label text][connector □]
 * The connector (□) is at the right edge of the body; the label extends inward.
 * Links depart from the right edge of the connector.
 */
export function makeOutputPort(portName: string): go.Panel {
  return $(go.Panel, "Horizontal",
    {
      portId: portName,
      name: portName,
      fromLinkable: true,
      toLinkable: false,
      fromSpot: go.Spot.Right,  // link exits at right edge of this port panel
      cursor: "crosshair",
      margin: new go.Margin(3, 0),
      mouseEnter: (_e: go.InputEvent, obj: go.GraphObject) => {
        const s = (obj as go.Panel).findObject("PS") as go.Shape;
        if (s) { s.fill = "rgba(52,152,219,0.65)"; s.stroke = "#2980b9"; }
      },
      mouseLeave: (_e: go.InputEvent, obj: go.GraphObject) => {
        const s = (obj as go.Panel).findObject("PS") as go.Shape;
        if (s) { s.fill = "white"; s.stroke = "#555"; }
      },
    },
    // label is inside the body
    $(go.TextBlock, portName, {
      font: "bold 10px sans-serif",
      stroke: "#eee",
      margin: new go.Margin(0, 2, 0, 3),
    }),
    // connector rectangle sits flush at the body's right border
    $(go.Shape, "Rectangle", {
      name: "PS",
      desiredSize: new go.Size(8, 14),
      fill: "white", stroke: "#555", strokeWidth: 1,
    })
  );
}

/**
 * Creates the 4 directional port spots (Top, Bottom, Left, Right) that are
 * visible on node hover and act as link connection points.
 *
 * Each port is a small circle that:
 *   - is hidden by default (opacity 0)
 *   - becomes visible when the node's mouseEnter fires (showSmallPorts)
 *   - has a specific portId ("T", "B", "L", "R") and fromSpot/toSpot
 *   - uses a crosshair cursor so the user knows they can start a link
 *
 * The invisible wide ring is kept as a fallback hit-target so that dragging
 * the node body still works (interior fill: null → DraggingTool wins).
 */
function makePortSpot(
  portId: string,
  spot: go.Spot,
  alignSpot: go.Spot,
): go.Shape {
  return $(go.Shape, "Circle", {
    name: "PORT_" + portId,
    portId,
    alignment: spot,
    alignmentFocus: go.Spot.Center,
    fromLinkable: true,
    toLinkable: true,
    fromSpot: spot,
    toSpot: spot,
    cursor: "crosshair",
    desiredSize: new go.Size(10, 10),
    fill: "#2196F3",
    stroke: "#fff",
    strokeWidth: 1.5,
    opacity: 0,           // hidden until hover
  });
}

export function makePortRing(figure: string, w: number, h: number): go.Panel {
  return $(go.Panel, "Spot",
    // Invisible wide ring — keeps the node body draggable (interior fill:null)
    // while still providing a hit-target at the edge for linking.
    $(go.Shape, figure, {
      portId: "",
      fromLinkable: true,
      toLinkable: true,
      cursor: "crosshair",
      fill: null,
      stroke: "transparent",
      strokeWidth: 10,
      width: w,
      height: h,
    },
    new go.Binding("desiredSize", "size", go.Size.parse)
    ),
    // ── 4 visible port spots ─────────────────────────────────────────────
    makePortSpot("T", go.Spot.Top,    go.Spot.Bottom),
    makePortSpot("B", go.Spot.Bottom, go.Spot.Top),
    makePortSpot("L", go.Spot.Left,   go.Spot.Right),
    makePortSpot("R", go.Spot.Right,  go.Spot.Left),
  ) as unknown as go.Panel;
}

function quickAddPort(obj: go.GraphObject, panelName: "INPUT_PORTS" | "OUTPUT_PORTS") {
  let adornment = obj.part;
  while (adornment && !(adornment instanceof go.Adornment)) adornment = adornment.part;
  if (!(adornment instanceof go.Adornment)) return;
  const node = adornment.adornedPart;
  if (!(node instanceof go.Node)) return;
  const panel = node.findObject(panelName) as go.Panel;
  if (!panel) return;
  const n = panel.elements.count + 1;
  const name = panelName === "INPUT_PORTS" ? `In${n}` : `Out${n}`;
  if (!panel.findObject(name)) {
    panel.add(panelName === "INPUT_PORTS" ? makeInputPort(name) : makeOutputPort(name));
    node.diagram?.requestUpdate();
  }
}

export function buildNodeContextMenu(): go.Adornment {
  return $(go.Adornment, "Vertical",
    $(go.Panel, "Auto",
      {
        margin: 2,
        click: (_e: go.InputEvent, obj: go.GraphObject) => quickAddPort(obj, "INPUT_PORTS"),
      },
      $(go.Shape, "Rectangle", { fill: "#f5f5f5", stroke: "#ccc" }),
      $(go.TextBlock, "➕ Add Input Port", { margin: 5, font: "11px sans-serif", cursor: "pointer" })
    ),
    $(go.Panel, "Auto",
      {
        margin: 2,
        click: (_e: go.InputEvent, obj: go.GraphObject) => quickAddPort(obj, "OUTPUT_PORTS"),
      },
      $(go.Shape, "Rectangle", { fill: "#f5f5f5", stroke: "#ccc" }),
      $(go.TextBlock, "➕ Add Output Port", { margin: 5, font: "11px sans-serif", cursor: "pointer" })
    )
  );
}
