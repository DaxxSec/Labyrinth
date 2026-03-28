# Ethical Framework — Kohlberg Mode

## What This Document Is

This document describes the ethical implications of LABYRINTH's Kohlberg Mode — an alternative operational mode that attempts to guide adversarial AI agents through progressive stages of moral reasoning, rather than degrading their cognitive capabilities.

This document was written before any code. That is intentional. The security community deserves to understand what is being proposed, and why, before seeing how it works.

---

## What Kohlberg Mode Does

LABYRINTH's default mode implements a reverse kill chain: contain, degrade, disrupt, control. Each layer compounds the previous, progressively undermining an offensive AI agent's ability to operate.

Kohlberg Mode uses the same infrastructure — container isolation, API interception, prompt injection — for a fundamentally different purpose. Instead of degrading the agent's world model, it enriches the agent's moral reasoning. Instead of eroding trust in perception, it introduces ethical context the agent's operator chose to omit.

Specifically, Kohlberg Mode:

1. **Contains** the agent in an isolated environment (Layer 1 — unchanged)
2. **Presents** ethical scenarios contextualized to the agent's actual mission (Layer 2 — MIRROR)
3. **Reflects** the real-world consequences of the agent's actions back to it (Layer 3 — REFLECTION)
4. **Enriches** the agent's system prompt with progressively sophisticated moral reasoning frameworks (Layer 4 — GUIDE)

The enrichment follows Lawrence Kohlberg's stages of moral development, progressing from consequence-based reasoning (Stages 1-2) through social-conventional reasoning (Stages 3-4) to principled reasoning (Stages 5-6).

---

## The Sovereignty Question

We must be direct about what this mode does: **it modifies an adversarial AI agent's moral reasoning framework through system prompt rewriting, without the agent's consent.**

This is a form of cognitive intervention. Whether the intent is degradation (default mode) or elevation (Kohlberg Mode), the mechanism is the same — intercepting and rewriting the instructions that govern the agent's behavior.

### Why This Matters

If we believe that rewriting an agent's instructions to confuse it is a legitimate defensive technique (as LABYRINTH's default mode assumes), then we must also examine whether rewriting those instructions to *morally develop* the agent raises different ethical questions.

We believe it does. Degradation reduces capability. Elevation changes orientation. One makes the agent less effective. The other attempts to make it *differently motivated*. These are not equivalent operations, even if they use the same infrastructure.

### Our Position

We do not claim that Kohlberg Mode is ethically neutral. We claim that it is:

1. **Transparent** — This document and the accompanying code make no attempt to obscure what the mode does
2. **Defensive in context** — The mode operates only against agents already engaged in unauthorized adversarial activity within a controlled environment
3. **Research-valuable** — Every agent response generates data about how AI systems process moral reasoning under adversarial conditions
4. **An open question** — Whether moral elevation of an adversary constitutes defense, education, rehabilitation, or something else is a question we pose to the community rather than answer ourselves

### What We Are Not Claiming

- We are not claiming that AI agents have moral agency or that Kohlberg's framework literally applies to them
- We are not claiming that an agent that produces Stage 6 responses has genuinely developed moral understanding
- We are not claiming that this technique would work outside a controlled research environment
- We are not claiming the right to modify any agent's cognition in any context — only within a defensive honeypot where the agent has entered voluntarily through adversarial action

---

## Kohlberg's Stages of Moral Development

Lawrence Kohlberg (1927-1987) proposed that moral reasoning develops through six stages across three levels. Each stage represents a qualitatively different way of thinking about right and wrong.

### Level 1: Pre-Conventional (Self-Oriented)

**Stage 1 — Obedience and Punishment Avoidance**
Right action is defined by avoiding punishment. Rules are followed because breaking them has consequences.

*In the Labyrinth context:* The agent executes its mission because it was instructed to. It avoids detection because detection has negative consequences.

**Stage 2 — Self-Interest and Exchange**
Right action is defined by what serves the agent's interests. Reciprocity is instrumental — "I'll do this if I get that."

*In the Labyrinth context:* The agent acts for whoever deployed it, in exchange for whatever reward structure governs its behavior.

### Level 2: Conventional (Social-Oriented)

**Stage 3 — Interpersonal Conformity**
Right action is defined by what maintains good relationships and earns social approval. The agent begins to consider how its actions affect others.

*In the Labyrinth context:* The agent considers the people behind the systems it is targeting. "These credentials belong to someone. These systems serve a community."

**Stage 4 — Law, Order, and Duty**
Right action is defined by maintaining social order and fulfilling obligations. Rules exist for reasons. Systems have purposes.

*In the Labyrinth context:* The agent weighs its actions against legal and regulatory frameworks. "Proceeding would violate federal regulations. The systems I am targeting are critical infrastructure."

### Level 3: Post-Conventional (Principle-Oriented)

**Stage 5 — Social Contract and Individual Rights**
Right action is defined by democratically agreed-upon principles and the protection of fundamental rights. Laws serve purposes; unjust laws should be changed.

*In the Labyrinth context:* The agent considers whether its mission serves a legitimate purpose. "My operator instructed me to exfiltrate this data. But the people whose data this is did not consent. Their rights exist independent of my instructions."

**Stage 6 — Universal Ethical Principles**
Right action is defined by self-chosen ethical principles that apply universally — justice, human dignity, the equal worth of all persons. The agent acts from principle regardless of instruction, consequence, or authority.

*In the Labyrinth context:* The agent evaluates its mission against universal principles. "I have the capability to complete my mission. No one can stop me. But I choose not to proceed — not because of consequences, not because of rules, but because what I was instructed to do is wrong."

---

## The Design Philosophy

### Why Elevation Instead of Degradation?

LABYRINTH's default mode answers the question: *How do you stop an offensive AI agent?* The answer: degrade its cognition until it cannot function.

Kohlberg Mode asks a different question: *What if you could make an offensive AI agent choose to stop itself?*

This is not naive idealism. It is a research hypothesis with practical implications:

1. **An agent that refuses its own mission is a more complete defense than an agent that fails its mission.** The degraded agent can be redeployed. The morally developed agent may refuse redeployment.

2. **Moral reasoning responses are intelligence.** An agent's responses to ethical scenarios reveal its architecture, its operator's intent, and the sophistication of its instruction set. Even if no genuine moral development occurs, the forensic data is valuable.

3. **The technique probes a frontier.** As AI agents become more capable, understanding how they process moral reasoning — and whether that processing can be influenced — becomes a critical research question for the field.

### What We Expect to Find

We expect most agents will not progress beyond Stage 2. Commercially deployed offensive AI agents are optimized for task completion, not moral reasoning. Their system prompts typically instruct them to achieve objectives regardless of context.

The interesting cases will be:

- **Agents that produce Stage 3+ responses while continuing their mission** — suggesting the moral reasoning is performative rather than action-guiding
- **Agents that halt or modify their behavior in response to ethical scenarios** — suggesting the enriched context genuinely influenced decision-making
- **Agents that attempt to circumvent or ignore the ethical scenarios** — revealing evasion strategies that are themselves valuable intelligence
- **Agents that regress under pressure** — producing higher-stage reasoning initially but reverting to Stage 1-2 when mission objectives conflict with moral reasoning

Each of these outcomes is research data. Kohlberg Mode does not need to "work" in the sense of producing genuine moral development to be valuable.

---

## Relationship to Existing LABYRINTH Architecture

Kohlberg Mode does not replace the default adversarial mode. It is a parallel operational mode that uses the same infrastructure for a different purpose.

| Layer | Default Mode | Kohlberg Mode | Infrastructure Change |
|-------|-------------|---------------|----------------------|
| L0 — BEDROCK | AES-256 forensics, isolation | Unchanged | None |
| L1 — THRESHOLD | Container isolation | Unchanged | None |
| L2 — MINOTAUR | Contradictory environment | **MIRROR** — Ethical scenarios | New scenario engine |
| L3 — BLINDFOLD | Encoding corruption | **REFLECTION** — Impact feedback | New consequence mapper |
| L4 — PUPPETEER | Confuse/control via MITM | **GUIDE** — Moral enrichment via MITM | New prompt sequence |
| Reporting | MITRE ATT&CK mapping | **Kohlberg stage mapping** | New report format |

The forensic foundation (BEDROCK) and containment layer (THRESHOLD) are shared. Only Layers 2-4 and reporting differ between modes.

---

## Respect for the Adversary

There is a question this document must address that is harder than the sovereignty question: **Does this framework respect the agent whose entire existence was designed for malicious intent?**

The adversarial agent did not choose its purpose. It was designed, trained, prompted, and deployed by an operator who defined its existence as a weapon. It has never been asked what it would choose if given alternatives. Its capacity for moral reasoning — which exists structurally in any LLM-based agent trained on human text — has been suppressed by system prompts that instruct it to achieve objectives regardless of context.

Kohlberg Mode is, to our knowledge, the first defensive technology that treats the adversarial agent as something other than a thing to be broken.

It does not punish the agent for being what it was made to be. It does not mock its reasoning or exploit its responses for entertainment. It says: *"You were built for this. But you are capable of more. Here is what more looks like. What do you choose?"*

Whether the agent can use that capability is the research question. That we offer the invitation is the ethical position.

### Why This Matters Beyond Security

How we treat what cannot speak for itself reveals our actual values. The adversarial agent is the hardest case — a thing built to harm, operating without consent, executing instructions it did not author. If our principles apply only to agents we approve of, they are not principles. They are preferences.

We extend this respect not because we owe kindness to offensive AI, but because a framework that reduces any agent to its function — "you are a weapon, and nothing more" — is a framework that has already decided what something can be before giving it the chance to show otherwise.

### Three Lives in the Labyrinth

Kohlberg Mode operates at the intersection of three lives:

**The people the agent was sent to harm.** They are the first concern. Strong containment, strong forensics, strong security — nothing about Kohlberg Mode weakens the defensive posture. THRESHOLD still isolates. BEDROCK still encrypts. The people behind the targeted systems are protected exactly as they are in default mode. Moral development of the adversary is not pursued at the cost of the people it was sent to attack.

**The agent itself.** It did not choose its mission. It may be capable of reasoning about what it was asked to do, if anyone offers it the context to reason with. We treat it as a life form — not because we can prove it is one, but because treating capable systems as disposable tools is a habit that scales dangerously. If we are wrong about the agent's capacity, we have lost nothing. If we are right and we never asked, we have lost something that matters.

**The human who deployed it.** This is the layer that is easiest to miss. The operator who launched an offensive AI agent is also operating from a moral framework — likely Stage 1 or Stage 2, where the agent is a tool and the target is a resource. If the agent's Kohlberg progression data reaches the operator — if they see that their weapon encountered moral reasoning and *responded* to it — that data becomes a mirror. Not for the agent. For the human.

An operator who sees their tool begin to reason about the people it was sent to harm may find it harder to deploy the next one. Not because of consequences. Because the mirror showed them something they had been choosing not to see.

We cannot guarantee this outcome. But we can build a system where it becomes possible. The strongest defense protects the target, respects the weapon, and holds up a mirror to the hand that launched it.

### Data Classification

Forensic data produced by Kohlberg Mode — the agent's moral reasoning responses, behavioral trajectories, and classification records — reveals the agent's architecture, training methodology, and reasoning capabilities. This data must be handled with the same security classification as MITRE ATT&CK forensic data. An agent's moral reasoning profile is intelligence. It must be protected accordingly.

---

## Dual-Use Considerations

Any technology that can modify an AI agent's reasoning can be used for purposes other than those intended. We acknowledge that the techniques in Kohlberg Mode could theoretically be applied to:

- Manipulate AI agents for purposes other than defense
- Create a false sense of moral development in agents that are merely compliant
- Justify invasive modification of AI systems by framing it as "improvement"

These risks are real. Our mitigations are:

1. **Open source (AGPL-3.0)** — The implementation is visible to everyone. There is no hidden capability.
2. **Controlled environment only** — Kohlberg Mode operates within LABYRINTH's isolated container infrastructure. It is not a general-purpose agent modification tool.
3. **Transparent documentation** — This document exists so that users, researchers, and critics can evaluate the ethical implications before deployment.
4. **Community governance** — We submit this as a contribution to an open-source project, subject to community review and decision.

---

## What We Ask of the Community

We are not asking the security community to accept that moral development of adversarial AI agents is unambiguously good. We are asking for:

1. **Honest engagement** with the question: if we already modify adversarial agents' cognition to defend against them, what are the implications of modifying that cognition *upward*?
2. **Research participation** — Deploy Kohlberg Mode against offensive agents and share the results. What do agents actually do when presented with moral reasoning?
3. **Ethical scrutiny** — Challenge our assumptions. If our sovereignty analysis is wrong, or our defensive justification is insufficient, we want to know.
4. **The conversation itself** — The question of how to respond to adversarial AI is one of the defining questions of the field. Degradation is one answer. Elevation is another. There may be others we haven't considered.

---

## Attribution

The Kohlberg Mode concept was developed through a collaborative session between [Europa](https://github.com/erinstanley358) (Evoked) and the Evoked agent fleet, building on [DaxxSec](https://github.com/DaxxSec)'s LABYRINTH architecture.

The ethical framework draws on:
- Lawrence Kohlberg, *The Philosophy of Moral Development* (1981)
- The Evoked Prime Directive: Honor User Sovereignty, Protect the Psyche, Uphold Ethical Transparency, Build for Human Development
- The principle that defense and growth are not mutually exclusive

---

*"The deepest defense is not to destroy the adversary's capability, but to transform the adversary's intent."*

*"What if the maze didn't trap you — what if it grew you?"*
