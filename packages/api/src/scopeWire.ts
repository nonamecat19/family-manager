import type { MessageInitShape } from "@bufbuild/protobuf";
import {
  PeriodGranularity,
  PeriodSchema,
  ScopeKind,
  ScopeSchema,
} from "@fm/sdk/finance/v1/finance_pb";

import { familyScope, type PeriodGranularityInput, type PeriodInput, type ScopeInput } from "./scope.ts";

export function toWireScope(scope: ScopeInput = familyScope): MessageInitShape<typeof ScopeSchema> {
  switch (scope.kind) {
    case "member":
      return { kind: ScopeKind.MEMBER, memberId: scope.memberId, accountId: "" };
    case "account":
      return { kind: ScopeKind.ACCOUNT, memberId: "", accountId: scope.accountId };
    case "family":
      return { kind: ScopeKind.FAMILY, memberId: "", accountId: "" };
  }
}

const WIRE_GRANULARITY: Record<PeriodInput["granularity"], PeriodGranularity> = {
  day: PeriodGranularity.DAY,
  week: PeriodGranularity.WEEK,
  month: PeriodGranularity.MONTH,
  year: PeriodGranularity.YEAR,
  custom: PeriodGranularity.CUSTOM,
};

export function toWirePeriod(period: PeriodInput): MessageInitShape<typeof PeriodSchema> {
  if (period.granularity === "custom") {
    return {
      granularity: PeriodGranularity.CUSTOM,
      anchor: period.range.from,
      range: { from: period.range.from, to: period.range.to },
    };
  }
  return { granularity: WIRE_GRANULARITY[period.granularity], anchor: period.anchor };
}

export function toWireGranularity(granularity: PeriodGranularityInput): PeriodGranularity {
  return WIRE_GRANULARITY[granularity];
}
