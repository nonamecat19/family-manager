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

const spacing = {
  xs: "4px",
  sm: "8px",
  md: "12px",
  lg: "16px",
  xl: "24px",
  "2xl": "32px",

  // numeric scale, n * 4px — covers fine-grained gaps that don't fit the named scale
  "1.25": "5px",
  "1.4": "5.6px",
  "2.75": "11px",
  "2.8": "11.2px",
  "3": "12px",
  "3.5": "14px",
  "4.2": "16.8px",
};

const radius = {
  sm: "6px",
  md: "10px",
  lg: "16px",
  full: "9999px",
};

module.exports = {
  theme: {
    extend: {
      colors,
      spacing,
      borderRadius: radius,
      fontSize: {
        caption: ["12px", { lineHeight: "16px" }],
        body: ["15px", { lineHeight: "22px" }],
        title: ["20px", { lineHeight: "26px" }],
        display: ["32px", { lineHeight: "38px" }],
        amount: ["28px", { lineHeight: "34px" }],
      },
    },
  },
};
