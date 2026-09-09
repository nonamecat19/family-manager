/**
 * @fm/ui — one design system, themed per app.
 *
 * Components read roles from the `<ThemeProvider>` above them (`surface`, `accent`,
 * `radius.md`) rather than NativeWind classes, which is what lets the same component render
 * native to Nocturne's tight dark system and Organic's soft light one. See @fm/theme for the
 * contract and the two themes.
 */

export { ThemeProvider, useTheme, type ThemeProviderProps } from "./theme.tsx";

export {
  Avatar,
  Badge,
  Body,
  Button,
  Caption,
  Card,
  CardHeader,
  Divider,
  EmptyState,
  Field,
  Heading,
  InlineAction,
  Loading,
  Muted,
  Row,
  Screen,
  TextLink,
  Title,
  type BadgeTone,
  type ButtonProps,
  type ButtonTone,
  type EmptyStateProps,
  type FieldProps,
} from "./primitives.tsx";

export { ErrorBoundary, type ErrorBoundaryProps } from "./ErrorBoundary.tsx";

export { PieChart, type PieChartProps, type PieSlice } from "./PieChart.tsx";

/* -------------------------------------------------------------------- family --- */

export {
  FamilyNameCard,
  FamilyMembersCard,
  FamilyInvitationsCard,
  LeaveFamilyCard,
  type FamilyNameCardProps,
  type FamilyNameCardStrings,
  type FamilyMembersCardProps,
  type FamilyMembersCardStrings,
  type FamilyInvitationsCardProps,
  type FamilyInvitationsCardStrings,
  type LeaveFamilyCardProps,
  type LeaveFamilyCardStrings,
} from "./family/FamilyManagement.tsx";

export type { FamilyMemberView, FamilyInvitationView, FamilyRole } from "./family/types.ts";
