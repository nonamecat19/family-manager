const colors = {
  primary: {
    DEFAULT: "#2F855A",
    fg: "#FFFFFF",
    muted: "#C6F6D5",
  },
  income: "#2F855A",
  expense: "#C53030",
  transfer: "#2B6CB0",

  bg: "#F7FAFC",
  surface: "#FFFFFF",
  border: "#E2E8F0",
  fg: "#1A202C",
  muted: "#718096",

  "bg-dark": "#12161C",
  "surface-dark": "#1A202C",
  "border-dark": "#2D3748",
  "fg-dark": "#F7FAFC",
  "muted-dark": "#A0AEC0",
};

// fine numeric scale, key n -> n * 4px, 1px resolution from 1px to 128px.
// gap/padding/margin brackets should use this instead of arbitrary [Npx] values.
const fineSpacing = {};
for (let px = 1; px <= 128; px += 1) {
  fineSpacing[String(px / 4)] = `${px}px`;
}

// the ×1.4 "roomy" variant seen in a few compact/expanded layouts — not on the 1px grain above.
const roomySpacing = {
  "0.7": "2.8px",
  "1.4": "5.6px",
  "2.1": "8.4px",
  "2.8": "11.2px",
  "4.2": "16.8px",
  "5.6": "22.4px",
};

const spacing = {
  xs: "4px",
  sm: "8px",
  md: "12px",
  lg: "16px",
  xl: "24px",
  "2xl": "32px",

  ...fineSpacing,
  ...roomySpacing,
};

const radius = {
  sm: "6px",
  md: "10px",
  lg: "16px",
  full: "9999px",
};

// fine numeric scale, key = px value, 0.5px resolution from 1px to 96px.
// text-[Npx] brackets should use this instead of arbitrary values.
const fineFontSize = {};
for (let halfPx = 2; halfPx <= 192; halfPx += 1) {
  const px = halfPx / 2;
  fineFontSize[String(px)] = `${px}px`;
}

const fontSize = {
  caption: ["12px", { lineHeight: "16px" }],
  body: ["15px", { lineHeight: "22px" }],
  title: ["20px", { lineHeight: "26px" }],
  display: ["32px", { lineHeight: "38px" }],
  amount: ["28px", { lineHeight: "34px" }],

  ...fineFontSize,
};

module.exports = {
  theme: {
    extend: {
      colors,
      spacing,
      borderRadius: radius,
      fontSize,
    },
  },
};
