const neutral = {
  100: "#f3f5fe",
  200: "#e4e7f5",
  300: "#cfd3e5",
  400: "#b2b6ca",
  500: "#9397ab",
  600: "#75798c",
  700: "#595d6c",
  800: "#3f424d",
  900: "#292b31",
};

const accent = {
  DEFAULT: "#9184d9",
  100: "#f5f4ff",
  200: "#e7e5fe",
  300: "#d2cefd",
  400: "#b5abfc",
  500: "#968ae0",
  600: "#796cbf",
  700: "#5d5294",
  800: "#423a6a",
  900: "#2b2741",
};

const bg = "#161826";
const surface = "#232532";
const text = "#e9e9ed";
const divider = "rgba(233,233,237,.16)";

const overspend = "#e5928a";

module.exports = {
  theme: {
    extend: {
      colors: {
        neutral,
        accent,
        overspend,
        primary: { DEFAULT: accent.DEFAULT, fg: bg, muted: accent[900] },
        bg,
        surface,
        border: neutral[800],
        divider,
        fg: text,
        muted: neutral[500],
        "bg-dark": bg,
        "surface-dark": surface,
        "border-dark": neutral[800],
        "fg-dark": text,
        "muted-dark": neutral[500],
        error: overspend,
        expense: overspend,
      },
      borderRadius: {
        sm: "4px",
        md: "8px",
        lg: "14px",
      },
    },
  },
};
