# 100 Stars Later: The One Flag That Changes Everything

**Or: How We Accidentally Solved Enterprise AI's Biggest Problem**

*Alice Vinogradova - February 2026*

---

Hey folks! Eight months ago I wrote about building a bridge between OData and MCP, claiming we'd just unlocked 15,000+ enterprise services for AI. Bold claim, right?

Today I'm back with proof: **100+ stars**, **25 forks**, **11 releases**, and one feature that might just be the most important thing we've built yet.

But first, let me tell you about the problem nobody saw coming.

---

## The "It Works... Until It Doesn't" Moment

Picture this. September 2025. A user connects their SAP Business Partner API to Claude. 485 entity sets. Every CRUD operation. Beautiful metadata. Everything we designed for.

Claude's response?

> "No API available."

Wait, what? The bridge worked perfectly! Tools were generated! Metadata parsed! What happened?

After days of debugging, the answer was humbling:

**LLMs have a practical limit of ~128 tools.** Beyond that, they experience "context rot" - degraded reasoning, tool selection failures, and eventually... giving up entirely.

We'd built a bridge that could generate thousands of tools. And that very capability was breaking it.

---

## The Numbers That Kept Me Up at Night

| Service | Tools Generated | Token Usage | Status |
|---------|-----------------|-------------|--------|
| Northwind (demo) | 157 | ~5,000 | Works! |
| SAP GWSAMPLE | 68 | ~2,000 | Works! |
| SAP Business Partner | 485 | ~37,000 | Fails |
| Two services combined | 550+ | ~45,000 | Definitely fails |

The more capable your OData service, the more likely our bridge would break Claude.

Ironic, isn't it?

---

## The Solution: One Tool to Rule Them All

Enter **Filipp Gnilyak** ([@vitalratel](https://github.com/vitalratel)).

While I was still scratching my head about the context rot problem, Filipp opened [PR #26](https://github.com/oisee/odata_mcp_go/pull/26) with a solution so elegant I wondered why I hadn't thought of it myself:

```bash
# Before: 485 tools, 37,000 tokens, Claude says "no"
./odata-mcp https://large-sap-service.com/odata/

# After: 1 tool, 900 tokens, Claude says "how can I help?"
./odata-mcp --universal https://large-sap-service.com/odata/
```

One flag. That's it.

But Filipp didn't stop there. His PR also fixed **10 open issues** in one sweep - GUID formatting, SAP namespace parsing, error handling, search capability detection, and more. The kind of contribution that transforms a project.

**Universal Tool Mode** collapses your entire OData service into a single intelligent tool. Instead of `filter_Products`, `get_Orders`, `create_Customers`... you get one `odata` tool that understands:

```json
{"action": "list", "target": "Products", "params": {"filter": "Price gt 100"}}
{"action": "get", "target": "Orders", "params": {"key": {"OrderID": 1}}}
{"action": "create", "target": "Customers", "params": {"data": {...}}}
```

### The Results

| Metric | Standard Mode | Universal Mode | Improvement |
|--------|---------------|----------------|-------------|
| Tools (SAP BP) | 485 | 1 | **99.8%** reduction |
| Tools (Northwind) | 157 | 1 | **99.4%** reduction |
| Token usage | ~37,000 | ~900 | **97.6%** reduction |
| Works with Claude? | Sometimes | Always | Priceless |

---

## "Why Didn't You Make It Default?"

Fair question. We could have. Here's why we didn't:

### 1. Backward Compatibility
Your existing configs still work. No surprises after upgrade.

### 2. Discoverability
Per-entity tools are self-documenting. When Claude sees `filter_Products_for_northwind`, it knows exactly what it can do. With universal mode, the LLM needs to understand the service structure first.

### 3. Small Services Work Great
If you have 20-30 tools, dedicated per-entity tools are perfect. No need to change.

### 4. Explicit Choice
This is a meaningful trade-off. You should consciously decide:
- **Standard mode**: Maximum discoverability, works for smaller services
- **Universal mode**: Maximum compatibility, essential for large services

The best features are opt-in, not forced.

---

## The Journey: v1.0.0 to v1.6.0

Let me take you through what we've built since June:

```
June 2025     v1.0.0   "The Bridge Exists!"
                       - OData v2 support
                       - Basic CRUD operations
                       - SAP CSRF token handling
                         │
July 2025     v1.2.x   "Community Takes Over"
                       - First external PR merged!
                       - OData v4 support
                       - Read-only modes
                       - Claude Code CLI compatibility
                         │
September    v1.5.x    "Enterprise Ready"
                       - Streamable HTTP transport
                       - AI Foundry compatibility
                       - SAP GUID auto-formatting
                       - Security hardening (TLS, tokens)
                         │
February     v1.6.0    "One Tool to Rule Them All"
2026                   - Universal tool mode
                       - MCP header forwarding
                       - 10 issues fixed in one release
                       - Zero open issues!
```

**11 releases. 49 commits. Zero open issues.**

---

## What Else Is New?

### MCP Header Forwarding
```bash
./odata-mcp --transport streamable-http --forward-mcp-headers https://api.com/odata/
```
Pass authentication headers dynamically per-request. Perfect for multi-tenant scenarios.

### AI Foundry Compatibility
```bash
./odata-mcp --protocol-version 2025-06-18 https://service.com/odata/
```
Works with Microsoft's AI Foundry out of the box.

### SAP GUID Auto-Formatting
No more `guid'...'` headaches. The bridge detects SAP services and formats GUIDs automatically.

### Security Hardening
HTTP transport now requires:
- Token authentication (`--mcp-token`)
- TLS for non-localhost (`--tls`)
- Explicit flag for all-interfaces binding

---

## The Community Effect

When I published the first article, I honestly didn't know if anyone would care. Enterprise software isn't exactly viral content.

But then:
- **100+ stars** - people actually use this
- **25 forks** - people build on this
- **External PRs** - people contribute to this
- **LinkedIn discussions** - people debate this

The SAP community especially embraced it. Turns out, a lot of folks were tired of clicking through Fiori tiles.

### Special Shoutouts

**Filipp Gnilyak** ([@vitalratel](https://github.com/vitalratel)) - The MVP of v1.6.0. His massive PR brought us universal tool mode and fixed 10 issues in one shot. This is what open source is about - someone sees a problem, builds the solution, and shares it with everyone. Filipp, you're a legend.

**Louis-Philippe Perras** ([@lpperras](https://github.com/lpperras)) - Contributed MCP header forwarding for dynamic authentication. Real-world use case, clean implementation.

**Holger Bruchelt** - For the podcast, beta testing, and being an early believer.

And everyone who:
- Filed detailed bug reports (you know who you are)
- Tested with production SAP systems (brave souls)
- Shared the project in their networks

---

## What's Next?

### Short Term
- Phase 2 Universal Tool: Include entity schemas in tool description
- Better error context for LLMs
- Performance optimizations

### Medium Term
- Multi-service routing (one tool, multiple services)
- Caching layer for metadata
- Webhook support for real-time updates

### Long Term
- Azure managed service? (Still gauging interest - drop a comment!)
- Visual query builder
- Natural language to OData translation

---

## Try It Right Now

### Installation (2 minutes)
1. Download from [releases](https://github.com/oisee/odata_mcp_go/releases/tag/v1.6.0)
2. Add to your PATH
3. Update `claude_desktop_config.json`:

```json
{
    "mcpServers": {
        "my-service": {
            "command": "odata-mcp",
            "args": [
                "--service", "https://your-odata-service.com/",
                "--universal",
                "--tool-shrink"
            ]
        }
    }
}
```

### The Test
Try this with your largest OData service:

```bash
# Check tool count in standard mode
./odata-mcp --trace https://your-service.com/odata/ | grep total_tools

# Check tool count in universal mode
./odata-mcp --trace --universal https://your-service.com/odata/ | grep total_tools
```

Share your before/after numbers in the comments!

---

## The "So What?" - Revisited

Eight months ago I wrote:

> "When 15,000+ production services become AI-accessible overnight, MCP stops being a dev toy and becomes the enterprise standard."

Today I'll add:

> "And when those services work reliably regardless of size, we're not just connecting systems. We're changing how enterprises think about AI integration."

The bridge was never about OData or MCP. It was about proving that enterprise AI integration doesn't have to be hard.

One `--universal` flag at a time.

---

## Thank You

To everyone who starred, forked, filed issues, submitted PRs, or just tried it out - thank you. This project exists because of you.

Here's to the next 100 stars.

---

**Links:**
- GitHub: [oisee/odata_mcp_go](https://github.com/oisee/odata_mcp_go)
- Release v1.6.0: [Download](https://github.com/oisee/odata_mcp_go/releases/tag/v1.6.0)
- Previous article: [I Built the Universal OData-MCP Bridge](https://linkedin.com/in/alice-vinogradova)

---

*P.S. - First prompt I'll run after publishing: "Claude, analyze the engagement on this article and draft responses to all comments."*

*P.P.S. - Yes, that actually works now.*

#EnterpriseAI #OData #MCP #OpenSource #SAP #100Stars
