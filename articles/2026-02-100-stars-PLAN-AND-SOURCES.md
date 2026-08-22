# Article Plan & Sources: "100 Stars Later"

## Article Metadata
- **Title:** 100 Stars Later: The One Flag That Changes Everything
- **Subtitle:** Or: How We Accidentally Solved Enterprise AI's Biggest Problem
- **Author:** Alice Vinogradova
- **Date:** February 2026
- **Type:** LinkedIn Article / Blog Post
- **Word Count Target:** ~2,000 words

---

## 1. FACTUAL GROUND

### Repository Statistics (Verified)
```json
{
  "stars": 102,
  "forks": 25,
  "watchers": 5,
  "open_issues": 1,
  "created": "2025-06-12T08:14:02Z"
}
```
**Source:** `gh api repos/oisee/odata_mcp_go`

### Release Timeline (11 Releases)
| Version | Date | Key Features |
|---------|------|--------------|
| v1.0.0 | 2025-06-23 | Initial release, OData v2 |
| v1.0.1 | 2025-06-27 | Bug fixes |
| v1.1.3 | 2025-07-01 | OData v4 support |
| v1.2.0 | 2025-07-04 | Read-only modes |
| v1.2.1 | 2025-07-04 | Claude Code compat |
| v1.2.3 | 2025-07-05 | Operation filtering |
| v1.3.0 | 2025-07-07 | Service hints system |
| v1.4.0 | 2025-07-20 | Security hardening |
| v1.5.0 | 2025-09-17 | Streamable HTTP transport |
| v1.5.1 | 2025-09-19 | AI Foundry compatibility |
| v1.6.0 | 2026-02-03 | Universal tool mode |

**Source:** `gh release list`

### Commit Count Since Article
- **49 commits** since June 23, 2025
- **Source:** `git log --oneline --since="2025-06-23" | wc -l`

---

## 2. KEY CONTRIBUTORS

### Filipp Gnilyak (@vitalratel)
- **Email:** f.gnilyak@corp.mail.ru
- **GitHub:** https://github.com/vitalratel
- **Contribution:** PR #26 - Universal tool mode + 10 issue fixes
- **Stats:** +5,625 additions, -2,382 deletions, 39 files changed
- **Merged:** 2026-02-03T23:26:10Z

**PR #26 Quote (from body):**
> "Addresses 10 open GitHub issues... Reduces tool count by 95-99% (157 → 1 tool). Reduces context usage by 96-98% (~37K → ~900 tokens). Fixes 'context rot' with large services."

**Source:** https://github.com/oisee/odata_mcp_go/pull/26

### Louis-Philippe Perras (@lpperras)
- **GitHub:** https://github.com/lpperras
- **Contribution:** PR #24 - MCP Header Forwarding
- **Merged:** 2026-02-03T23:26:10Z

**PR #24 Quote:**
> "This PR adds a new feature allows the MCP server to forward any headers sent by the MCP client to the OData endpoint. This can be useful if you are using a secured OData endpoint with custom headers..."

**Source:** https://github.com/oisee/odata_mcp_go/pull/24

---

## 3. THE PROBLEM (Issue #14)

### Original Issue
- **Title:** "Multiple services at the same time - Claude is stuck if activated contextually"
- **Author:** Gabriele Rendina (@Raistlin82)
- **Created:** 2025-08-11T05:21:52Z
- **Closed:** 2026-02-03T23:33:42Z

**Quote from Issue:**
> "When defining two OData at the same time in the configuration, for a S4H Cloud solution, the MCP is not responding. If I isolate them it's working (one by one). Can you please check if it's related to the number of tools overall? API_SALES_ORDER_SRV gives: 113 tools while API_BUSINESS_PARTNER are 372."

**Source:** https://github.com/oisee/odata_mcp_go/issues/14

### Root Cause Analysis (from docs/008)
> "User configured two OData services totaling 485 tools. Claude reported 'no API available.' Root cause: LLMs have practical limits on tool count (~128). Exceeding this causes: Context rot (degraded reasoning), Tool selection failures, High token consumption (~14,000 tokens for schemas)"

**Source:** `docs/008-issue-14-universal-tool-architecture.md`

---

## 4. THE SOLUTION (Universal Tool Mode)

### Measured Results (from PR #26)

| Service | Standard Mode | Universal Mode | Reduction |
|---------|---------------|----------------|-----------|
| SAP GWSAMPLE_BASIC | 68 tools, ~16K tokens | 1 tool, ~765 tokens | 96% |
| Northwind v2 | 157 tools, ~37K tokens | 1 tool, ~912 tokens | 98% |

**Source:** PR #26 body

### Live Test Results (from our session)
```
Standard mode:  157 tools (total_tools: 157)
Universal mode:   1 tool  (total_tools: 1)
Reduction: 99.4%
```
**Source:** `./odata-mcp --trace --universal https://services.odata.org/V2/Northwind/Northwind.svc/`

### Commit Messages (Filipp's commits)

**Commit 1:** `fix: Address multiple GitHub issues and refactor client package`
> "Issues Fixed: #16: GUID handling, #18: $search capability, #22: BaseType support, #25: Windows build. Refactoring: Split client.go (900 lines) into focused modules..."

**Commit 2:** `feat: Add universal tool mode to solve tool explosion (Issue #14)`
> "Implements Phase 1 of universal tool architecture. Add --universal flag for single-tool mode instead of per-entity tools. Reduces tool count by 95-99%..."

**Commit 3:** `feat: Make universal tool mode the default`
> "Changes default from --universal=false to --universal=true. This reduces tool count from 157 to 1 for Northwind service, eliminating context rot for large OData services."

**Source:** `gh pr view 26 --json commits`

---

## 5. ISSUES FIXED IN v1.6.0

| # | Title | Fixed By |
|---|-------|----------|
| 12 | SAP OData Integration - No tools | PR #26 (Filipp) |
| 13 | --max-items 99999 crash | PR #26 (Filipp) |
| 14 | Multiple services - Claude stuck | PR #26 (Filipp) |
| 16 | GUID formatting | PR #26 (Filipp) |
| 17 | Timeout instead of error | PR #26 (Filipp) |
| 18 | Wildcard search fails | PR #26 (Filipp) |
| 19 | Timeout hides SAP error | PR #26 (Filipp) |
| 22 | BaseType not exposed | PR #26 (Filipp) |
| 23 | Header handling | PR #24 (Louis-Philippe) |
| 25 | Windows .exe | PR #26 (Filipp) |

**Source:** `gh issue list --state closed`

---

## 6. ARTICLE STRUCTURE

### Opening Hook (200 words)
- Callback to original article (June 2025)
- Proof: 100 stars, 25 forks, 11 releases
- Tease: "one feature that changes everything"

### The Problem (300 words)
- September 2025 user story (Issue #14)
- Quote from Gabriele Rendina
- "No API available" mystery
- Reveal: Context rot at ~128 tools

### The Numbers (200 words)
- Table: Tools vs Status
- "The more capable, the more likely to break"
- Irony angle

### The Solution (400 words)
- **Introduce Filipp** - the hero
- PR #26 stats: +5,625/-2,382, 39 files
- Before/after code example
- Results table from PR

### Why Opt-In (200 words)
- Backward compatibility
- Discoverability trade-off
- Small services work great
- Explicit choice philosophy

### The Journey Timeline (200 words)
- Visual: v1.0.0 → v1.6.0
- Key milestones
- 11 releases in 8 months

### Community (200 words)
- Filipp spotlight
- Louis-Philippe credit
- Other contributors
- SAP community adoption

### What's Next (100 words)
- Phase 2 teaser
- Azure service interest gauge

### Call to Action (100 words)
- Try --universal
- Share before/after
- Star the repo

---

## 7. KEY LINKS

### Repository
- **Main:** https://github.com/oisee/odata_mcp_go
- **Release v1.6.0:** https://github.com/oisee/odata_mcp_go/releases/tag/v1.6.0
- **PR #26:** https://github.com/oisee/odata_mcp_go/pull/26
- **PR #24:** https://github.com/oisee/odata_mcp_go/pull/24
- **Issue #14:** https://github.com/oisee/odata_mcp_go/issues/14

### Documentation
- **Universal Tool Architecture:** `docs/008-issue-14-universal-tool-architecture.md`
- **Security Analysis:** `docs/004-security-analysis-http-transport.md`
- **README:** https://github.com/oisee/odata_mcp_go/blob/main/README.md

### Previous Articles
- **Original Article:** "I Built the Universal OData ↔ MCP Bridge" (June 23, 2025)
- **Fiori Article:** "I Skipped Fiori (And I'd Do It Again)" (June 26, 2025)

---

## 8. QUOTABLE SNIPPETS

### For LinkedIn Post
> "485 tools → 1 tool. 37,000 tokens → 900 tokens. One flag: --universal"

### For Twitter/X
> "We accidentally solved enterprise AI's biggest problem. LLMs choke at ~128 tools. SAP services generate 400+. Solution? One tool to rule them all. #MCP #OData #EnterpriseAI"

### For The Hero Moment
> "While I was still scratching my head, Filipp opened PR #26 with a solution so elegant I wondered why I hadn't thought of it myself."

### For The Philosophy
> "The best features are opt-in, not forced."

---

## 9. IMAGES TO CREATE

1. **Stats Infographic:** Stars/Forks/Releases growth
2. **Timeline:** v1.0.0 → v1.6.0 journey
3. **Before/After:** Tool explosion → Single tool
4. **Token Comparison:** Bar chart showing 97% reduction

---

## 10. HASHTAGS

```
#EnterpriseAI #OData #MCP #OpenSource #SAP #100Stars #AIIntegration #Claude #LLM
```

---

## 11. PUBLISH CHECKLIST

- [ ] Proofread article
- [ ] Create infographics
- [ ] Tag contributors (Filipp, Louis-Philippe, Holger)
- [ ] Schedule for optimal time (Tuesday 9 AM?)
- [ ] Prepare follow-up comments with links
- [ ] Cross-post to Twitter/X
- [ ] Share in SAP community forums
