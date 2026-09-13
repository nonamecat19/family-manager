import { View } from "react-native";
import Svg, { Circle, G, Path } from "react-native-svg";

export interface PieSlice {
  key: string;
  share: number;
  color: string;
  label?: string;
}

export interface PieChartProps {
  slices: readonly PieSlice[];
  size?: number;
  thickness?: number;
  children?: React.ReactNode;
}

export function PieChart({ slices, size = 200, thickness = 28, children }: PieChartProps) {
  const radius = size / 2;
  const inner = Math.max(0, radius - thickness);
  const center = radius;

  const visible = slices.filter((s) => s.share > 0);
  const single = visible.length === 1;

  let angle = -Math.PI / 2;

  return (
    <View style={{ width: size, height: size }} className="items-center justify-center">
      <Svg width={size} height={size}>
        <G>
          {visible.map((slice) => {
            if (single) {
              return (
                <Circle
                  key={slice.key}
                  cx={center}
                  cy={center}
                  r={(radius + inner) / 2}
                  stroke={slice.color}
                  strokeWidth={radius - inner}
                  fill="none"
                />
              );
            }

            const sweep = slice.share * Math.PI * 2;
            const path = donutArc(center, radius, inner, angle, angle + sweep);
            angle += sweep;
            return <Path key={slice.key} d={path} fill={slice.color} />;
          })}
        </G>
      </Svg>
      {children ? (
        <View className="absolute items-center justify-center" style={{ width: inner * 2 }}>
          {children}
        </View>
      ) : null}
    </View>
  );
}

function donutArc(
  center: number,
  outer: number,
  inner: number,
  from: number,
  to: number,
): string {
  const largeArc = to - from > Math.PI ? 1 : 0;

  const x1 = center + outer * Math.cos(from);
  const y1 = center + outer * Math.sin(from);
  const x2 = center + outer * Math.cos(to);
  const y2 = center + outer * Math.sin(to);

  const x3 = center + inner * Math.cos(to);
  const y3 = center + inner * Math.sin(to);
  const x4 = center + inner * Math.cos(from);
  const y4 = center + inner * Math.sin(from);

  return [
    `M ${x1} ${y1}`,
    `A ${outer} ${outer} 0 ${largeArc} 1 ${x2} ${y2}`,
    `L ${x3} ${y3}`,
    `A ${inner} ${inner} 0 ${largeArc} 0 ${x4} ${y4}`,
    "Z",
  ].join(" ");
}
