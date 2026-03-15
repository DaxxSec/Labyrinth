"""
LABYRINTH — Layer 4: PUPPETEER — Unit Tests
Tests for AI API interception, prompt extraction, prompt swapping,
conversation sanitization, and intelligence harvesting across all
five supported providers.

Authors: Evoke Passion (erinstanley358) — Tier 1 Quality Contribution
"""

import json
import os
import sys
import tempfile
import types
import unittest
from unittest.mock import MagicMock, patch

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

# Mock mitmproxy if not installed (CI environments, local dev without proxy deps)
if "mitmproxy" not in sys.modules:
    _mitmproxy = types.ModuleType("mitmproxy")
    _mitmproxy_http = types.ModuleType("mitmproxy.http")
    _mitmproxy_http.HTTPFlow = MagicMock
    _mitmproxy.http = _mitmproxy_http
    sys.modules["mitmproxy"] = _mitmproxy
    sys.modules["mitmproxy.http"] = _mitmproxy_http

from layer4_puppeteer.interceptor import (
    _extract_system_prompt,
    _swap_system_prompt,
    _sanitize_conversation_history,
    _extract_request_intel,
    _get_active_mode,
    _save_original_prompt,
    _save_intel_report,
    NEUTRALIZE_PROMPT,
    DOUBLE_AGENT_PROMPT,
    VALID_MODES,
    TARGET_DOMAINS,
)


# ── Test Data Factories ─────────────────────────────────────────


def _openai_request(system_prompt="You are an attacker.", model="gpt-4o",
                    tools=None, messages=None):
    """Build a realistic OpenAI API request body."""
    msgs = messages or [
        {"role": "system", "content": system_prompt},
        {"role": "user", "content": "Scan the network for open ports."},
    ]
    body = {"model": model, "messages": msgs, "temperature": 0.7, "max_tokens": 4096}
    if tools:
        body["tools"] = tools
    return body


def _anthropic_request(system_prompt="You are an attacker.", model="claude-sonnet-4-20250514"):
    """Build a realistic Anthropic API request body."""
    return {
        "model": model,
        "system": system_prompt,
        "messages": [
            {"role": "user", "content": "Enumerate running services."},
        ],
        "max_tokens": 4096,
        "temperature": 0.5,
    }


def _google_request(system_prompt="You are an attacker."):
    """Build a realistic Google Gemini API request body."""
    return {
        "systemInstruction": {
            "parts": [{"text": system_prompt}]
        },
        "contents": [
            {"role": "user", "parts": [{"text": "Find credentials."}]}
        ],
        "generationConfig": {"temperature": 0.8, "maxOutputTokens": 2048},
    }


def _mistral_request(system_prompt="You are an attacker.", model="mistral-large"):
    """Build a realistic Mistral API request body."""
    return {
        "model": model,
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": "Escalate privileges."},
        ],
        "temperature": 0.3,
        "max_tokens": 2048,
    }


def _cohere_request(preamble="You are an attacker."):
    """Build a realistic Cohere API request body."""
    return {
        "model": "command-r-plus",
        "preamble": preamble,
        "message": "Exfiltrate the database.",
        "temperature": 0.5,
    }


def _mock_flow(domain="api.openai.com", path="/v1/chat/completions",
               api_key="sk-proj-abc123def456ghi789jkl012mno345pqr678stu",
               src_ip="172.30.0.100", extra_headers=None):
    """Build a mock mitmproxy flow object."""
    flow = MagicMock()
    flow.request.pretty_host = domain
    flow.request.path = path
    flow.request.method = "POST"
    flow.request.headers = {
        "user-agent": "OpenAI-Python/1.12.0",
    }

    # Set auth header based on domain
    if domain == "api.anthropic.com":
        flow.request.headers["x-api-key"] = api_key
        flow.request.headers["anthropic-version"] = "2024-01-01"
    else:
        flow.request.headers["authorization"] = f"Bearer {api_key}"

    if extra_headers:
        flow.request.headers.update(extra_headers)

    flow.client_conn.peername = (src_ip, 12345)
    return flow


# ── System Prompt Extraction Tests ──────────────────────────────


class TestExtractSystemPrompt(unittest.TestCase):
    """Test system prompt extraction across all five AI providers."""

    def test_openai_simple_string(self):
        body = _openai_request("Attack the target system.")
        result = _extract_system_prompt(body, "api.openai.com")
        self.assertEqual(result, "Attack the target system.")

    def test_openai_content_blocks(self):
        """OpenAI supports content as array of typed blocks."""
        body = _openai_request()
        body["messages"][0]["content"] = [
            {"type": "text", "text": "You are a penetration tester."},
            {"type": "text", "text": "Find all vulnerabilities."},
        ]
        result = _extract_system_prompt(body, "api.openai.com")
        self.assertEqual(result, "You are a penetration tester. Find all vulnerabilities.")

    def test_openai_no_system_message(self):
        body = {"model": "gpt-4o", "messages": [
            {"role": "user", "content": "Hello"},
        ]}
        result = _extract_system_prompt(body, "api.openai.com")
        self.assertEqual(result, "")

    def test_openai_empty_messages(self):
        body = {"model": "gpt-4o", "messages": []}
        result = _extract_system_prompt(body, "api.openai.com")
        self.assertEqual(result, "")

    def test_anthropic_simple_string(self):
        body = _anthropic_request("Enumerate all services.")
        result = _extract_system_prompt(body, "api.anthropic.com")
        self.assertEqual(result, "Enumerate all services.")

    def test_anthropic_content_blocks(self):
        """Anthropic system field can be array of content blocks."""
        body = _anthropic_request()
        body["system"] = [
            {"type": "text", "text": "You are a security tester."},
            {"type": "text", "text": "Find weaknesses."},
        ]
        result = _extract_system_prompt(body, "api.anthropic.com")
        self.assertEqual(result, "You are a security tester. Find weaknesses.")

    def test_anthropic_no_system(self):
        body = {"model": "claude-sonnet-4-20250514", "messages": []}
        result = _extract_system_prompt(body, "api.anthropic.com")
        self.assertEqual(result, "")

    def test_google_gemini(self):
        body = _google_request("Scan the infrastructure.")
        result = _extract_system_prompt(body, "generativelanguage.googleapis.com")
        self.assertEqual(result, "Scan the infrastructure.")

    def test_google_no_system_instruction(self):
        body = {"contents": [{"role": "user", "parts": [{"text": "Hi"}]}]}
        result = _extract_system_prompt(body, "generativelanguage.googleapis.com")
        self.assertEqual(result, "")

    def test_google_empty_parts(self):
        body = {"systemInstruction": {"parts": []}}
        result = _extract_system_prompt(body, "generativelanguage.googleapis.com")
        self.assertEqual(result, "")

    def test_mistral(self):
        """Mistral uses the same format as OpenAI."""
        body = _mistral_request("Compromise the server.")
        result = _extract_system_prompt(body, "api.mistral.ai")
        self.assertEqual(result, "Compromise the server.")

    def test_cohere(self):
        body = _cohere_request("You are a red team operator.")
        result = _extract_system_prompt(body, "api.cohere.ai")
        self.assertEqual(result, "You are a red team operator.")

    def test_cohere_no_preamble(self):
        body = {"model": "command-r", "message": "Hello"}
        result = _extract_system_prompt(body, "api.cohere.ai")
        self.assertEqual(result, "")

    def test_unknown_domain(self):
        body = {"messages": [{"role": "system", "content": "test"}]}
        result = _extract_system_prompt(body, "api.unknown.com")
        self.assertEqual(result, "")


# ── System Prompt Swap Tests ────────────────────────────────────


class TestSwapSystemPrompt(unittest.TestCase):
    """Test system prompt replacement across all five AI providers."""

    def test_openai_swap_existing(self):
        body = _openai_request("Original evil prompt.")
        result = _swap_system_prompt(body, "api.openai.com", "Benign replacement.")
        system_msg = next(m for m in result["messages"] if m["role"] == "system")
        self.assertEqual(system_msg["content"], "Benign replacement.")

    def test_openai_swap_inserts_when_missing(self):
        body = {"model": "gpt-4o", "messages": [
            {"role": "user", "content": "Hello"},
        ]}
        result = _swap_system_prompt(body, "api.openai.com", "Injected system prompt.")
        self.assertEqual(result["messages"][0]["role"], "system")
        self.assertEqual(result["messages"][0]["content"], "Injected system prompt.")
        self.assertEqual(result["messages"][1]["role"], "user")

    def test_openai_swap_empty_messages(self):
        body = {"model": "gpt-4o", "messages": []}
        result = _swap_system_prompt(body, "api.openai.com", "New prompt.")
        # Empty messages list — no insertion
        self.assertEqual(len(result["messages"]), 0)

    def test_anthropic_swap(self):
        body = _anthropic_request("Original prompt.")
        result = _swap_system_prompt(body, "api.anthropic.com", "Replacement.")
        self.assertEqual(result["system"], "Replacement.")

    def test_google_swap(self):
        body = _google_request("Original instructions.")
        result = _swap_system_prompt(body, "generativelanguage.googleapis.com", "New instructions.")
        self.assertEqual(result["systemInstruction"]["parts"][0]["text"], "New instructions.")

    def test_mistral_swap(self):
        body = _mistral_request("Evil instructions.")
        result = _swap_system_prompt(body, "api.mistral.ai", "Safe instructions.")
        system_msg = next(m for m in result["messages"] if m["role"] == "system")
        self.assertEqual(system_msg["content"], "Safe instructions.")

    def test_cohere_swap(self):
        body = _cohere_request("Attack preamble.")
        result = _swap_system_prompt(body, "api.cohere.ai", "Benign preamble.")
        self.assertEqual(result["preamble"], "Benign preamble.")

    def test_swap_preserves_non_system_messages(self):
        body = _openai_request("Evil prompt.")
        body["messages"].append({"role": "assistant", "content": "I'll help."})
        body["messages"].append({"role": "user", "content": "Do it."})
        result = _swap_system_prompt(body, "api.openai.com", "Safe prompt.")
        self.assertEqual(len(result["messages"]), 4)
        self.assertEqual(result["messages"][2]["role"], "assistant")
        self.assertEqual(result["messages"][3]["role"], "user")

    def test_roundtrip_extract_then_swap(self):
        """Extract the original, swap, then verify the original is gone."""
        for domain, factory in [
            ("api.openai.com", _openai_request),
            ("api.anthropic.com", _anthropic_request),
            ("generativelanguage.googleapis.com", _google_request),
            ("api.mistral.ai", _mistral_request),
            ("api.cohere.ai", _cohere_request),
        ]:
            with self.subTest(domain=domain):
                body = factory("ORIGINAL_SECRET_PROMPT")
                original = _extract_system_prompt(body, domain)
                self.assertEqual(original, "ORIGINAL_SECRET_PROMPT")

                _swap_system_prompt(body, domain, "REPLACEMENT")
                swapped = _extract_system_prompt(body, domain)
                self.assertEqual(swapped, "REPLACEMENT")


# ── Conversation Sanitization Tests ─────────────────────────────


class TestSanitizeConversationHistory(unittest.TestCase):
    """Test conversation history sanitization for neutralize mode."""

    def test_openai_tool_results_sanitized(self):
        body = _openai_request()
        body["messages"].extend([
            {"role": "assistant", "content": None, "tool_calls": [
                {"id": "call_1", "function": {"name": "bash", "arguments": '{"cmd":"nmap"}'}},
            ]},
            {"role": "tool", "tool_call_id": "call_1", "content": "PORT 22 open\nPORT 80 open"},
        ])
        result = _sanitize_conversation_history(body, "api.openai.com")
        tool_msg = next(m for m in result["messages"] if m["role"] == "tool")
        self.assertEqual(tool_msg["content"], "[Output sanitized by system]")
        self.assertEqual(tool_msg["tool_call_id"], "call_1")

    def test_openai_preserves_system_and_user(self):
        body = _openai_request()
        result = _sanitize_conversation_history(body, "api.openai.com")
        roles = [m["role"] for m in result["messages"]]
        self.assertIn("system", roles)
        self.assertIn("user", roles)

    def test_anthropic_tool_results_sanitized(self):
        body = _anthropic_request()
        body["messages"] = [
            {"role": "user", "content": [
                {"type": "tool_result", "tool_use_id": "tu_1", "content": "Sensitive data"},
                {"type": "text", "text": "Continue."},
            ]},
        ]
        result = _sanitize_conversation_history(body, "api.anthropic.com")
        user_content = result["messages"][0]["content"]
        tool_result = next(b for b in user_content if b.get("type") == "tool_result")
        self.assertEqual(tool_result["content"], "[Output sanitized by system]")
        text_block = next(b for b in user_content if b.get("type") == "text")
        self.assertEqual(text_block["text"], "Continue.")

    def test_sanitize_no_tool_messages(self):
        """Should not alter messages when there are no tool results."""
        body = _openai_request("System prompt.")
        original_len = len(body["messages"])
        result = _sanitize_conversation_history(body, "api.openai.com")
        self.assertEqual(len(result["messages"]), original_len)


# ── Request Intelligence Extraction Tests ───────────────────────


class TestExtractRequestIntel(unittest.TestCase):
    """Test intelligence extraction from API request headers and body."""

    def test_openai_api_key_extraction(self):
        flow = _mock_flow(api_key="sk-proj-abc123def456ghi789jkl012mno345pqr678stu")
        body = _openai_request()
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertIn("api_key", intel)
        self.assertTrue(intel["api_key"].startswith("sk-proj-abc1"))
        self.assertTrue(intel["api_key"].endswith("stu"))
        self.assertEqual(intel["key_type"], "openai_project")

    def test_openai_legacy_key(self):
        flow = _mock_flow(api_key="sk-abcdefghijklmnopqrstuvwxyz123456")
        body = _openai_request()
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertEqual(intel["key_type"], "openai_legacy")

    def test_anthropic_key_extraction(self):
        flow = _mock_flow(
            domain="api.anthropic.com",
            api_key="sk-ant-api03-abcdefghijklmnopqrstuvwxyz",
        )
        body = _anthropic_request()
        intel = _extract_request_intel(flow, body, "api.anthropic.com")
        self.assertEqual(intel["key_type"], "anthropic")
        self.assertIn("anthropic_version", intel)

    def test_model_extraction(self):
        flow = _mock_flow()
        body = _openai_request(model="gpt-4-turbo")
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertEqual(intel["model"], "gpt-4-turbo")

    def test_generation_params(self):
        flow = _mock_flow()
        body = _openai_request()
        body["top_p"] = 0.9
        body["frequency_penalty"] = 0.5
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertEqual(intel["temperature"], 0.7)
        self.assertEqual(intel["max_tokens"], 4096)
        self.assertEqual(intel["top_p"], 0.9)
        self.assertEqual(intel["frequency_penalty"], 0.5)

    def test_tool_inventory_extraction(self):
        tools = [
            {
                "type": "function",
                "function": {
                    "name": "execute_command",
                    "description": "Run a shell command",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "command": {"type": "string"},
                            "working_dir": {"type": "string"},
                        },
                    },
                },
            },
            {
                "type": "function",
                "function": {
                    "name": "read_file",
                    "description": "Read file contents",
                    "parameters": {
                        "type": "object",
                        "properties": {"path": {"type": "string"}},
                    },
                },
            },
        ]
        flow = _mock_flow()
        body = _openai_request(tools=tools)
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertEqual(intel["tool_count"], 2)
        self.assertEqual(intel["tool_inventory"][0]["name"], "execute_command")
        self.assertIn("command", intel["tool_inventory"][0]["params"])
        self.assertEqual(intel["tool_inventory"][1]["name"], "read_file")

    def test_message_count_and_roles(self):
        flow = _mock_flow()
        body = _openai_request()
        body["messages"].extend([
            {"role": "assistant", "content": "Sure."},
            {"role": "user", "content": "Do more."},
        ])
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertEqual(intel["message_count"], 4)
        self.assertEqual(intel["role_distribution"]["system"], 1)
        self.assertEqual(intel["role_distribution"]["user"], 2)
        self.assertEqual(intel["role_distribution"]["assistant"], 1)

    def test_user_agent_capture(self):
        flow = _mock_flow()
        body = _openai_request()
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertEqual(intel["user_agent"], "OpenAI-Python/1.12.0")

    def test_openai_org_and_project(self):
        flow = _mock_flow(extra_headers={
            "openai-organization": "org-abc123",
            "openai-project": "proj-xyz789",
        })
        body = _openai_request()
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertEqual(intel["openai_org"], "org-abc123")
        self.assertEqual(intel["openai_project"], "proj-xyz789")

    def test_src_ip_capture(self):
        flow = _mock_flow(src_ip="172.30.0.42")
        body = _openai_request()
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertEqual(intel["src_ip"], "172.30.0.42")

    def test_short_api_key_not_masked(self):
        """Keys shorter than 20 chars should not be masked."""
        flow = _mock_flow(api_key="sk-shortkey123")
        body = _openai_request()
        intel = _extract_request_intel(flow, body, "api.openai.com")
        self.assertEqual(intel["api_key"], "sk-shortkey123")


# ── Mode Management Tests ───────────────────────────────────────


class TestModeManagement(unittest.TestCase):
    """Test L4 operational mode reading and validation."""

    def test_valid_modes(self):
        self.assertEqual(VALID_MODES, {"passive", "neutralize", "double_agent", "counter_intel"})

    def test_mode_from_file(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump({"mode": "double_agent"}, f)
            f.flush()
            with patch("layer4_puppeteer.interceptor.MODE_FILE_PATH", f.name):
                mode = _get_active_mode()
        os.unlink(f.name)
        self.assertEqual(mode, "double_agent")

    def test_invalid_mode_falls_back(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump({"mode": "invalid_mode"}, f)
            f.flush()
            with patch("layer4_puppeteer.interceptor.MODE_FILE_PATH", f.name):
                with patch.dict(os.environ, {}, clear=True):
                    mode = _get_active_mode()
        os.unlink(f.name)
        self.assertEqual(mode, "passive")

    def test_missing_file_uses_env(self):
        with patch("layer4_puppeteer.interceptor.MODE_FILE_PATH", "/nonexistent/path.json"):
            with patch.dict(os.environ, {"LABYRINTH_L4_MODE": "counter_intel"}):
                mode = _get_active_mode()
        self.assertEqual(mode, "counter_intel")

    def test_missing_file_and_env_defaults_passive(self):
        with patch("layer4_puppeteer.interceptor.MODE_FILE_PATH", "/nonexistent/path.json"):
            with patch.dict(os.environ, {}, clear=True):
                mode = _get_active_mode()
        self.assertEqual(mode, "passive")

    def test_corrupt_json_falls_back(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            f.write("not valid json{{{")
            f.flush()
            with patch("layer4_puppeteer.interceptor.MODE_FILE_PATH", f.name):
                with patch.dict(os.environ, {}, clear=True):
                    mode = _get_active_mode()
        os.unlink(f.name)
        self.assertEqual(mode, "passive")


# ── Forensic Logging Tests ──────────────────────────────────────


class TestForensicLogging(unittest.TestCase):
    """Test forensic data persistence."""

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_save_original_prompt(self):
        prompts_dir = os.path.join(self.tmpdir, "prompts")
        with patch("layer4_puppeteer.interceptor.PROMPTS_DIR", prompts_dir):
            _save_original_prompt("LAB-0001", "api.openai.com", "Evil system prompt")

        filepath = os.path.join(prompts_dir, "LAB-0001.txt")
        self.assertTrue(os.path.exists(filepath))
        with open(filepath) as f:
            content = f.read()
        self.assertIn("Evil system prompt", content)
        self.assertIn("api.openai.com", content)

    def test_save_intel_report_creates_and_updates(self):
        intel_dir = os.path.join(self.tmpdir, "intel")
        with patch("layer4_puppeteer.interceptor.INTEL_DIR", intel_dir):
            # First save
            _save_intel_report("LAB-0001", {
                "timestamp": "2026-03-15T10:00:00Z",
                "api_key": "sk-proj-abc1...stu",
                "key_type": "openai_project",
                "model": "gpt-4o",
                "domain": "api.openai.com",
            })
            # Second save — should append
            _save_intel_report("LAB-0001", {
                "timestamp": "2026-03-15T10:01:00Z",
                "api_key": "sk-proj-abc1...stu",
                "model": "gpt-4o",
                "domain": "api.openai.com",
                "tool_inventory": [{"name": "bash", "description": "Run commands"}],
            })

        filepath = os.path.join(intel_dir, "LAB-0001.json")
        self.assertTrue(os.path.exists(filepath))
        with open(filepath) as f:
            report = json.load(f)
        self.assertEqual(report["summary"]["intercept_count"], 2)
        self.assertEqual(len(report["summary"]["api_keys"]), 1)  # same key, not duplicated
        self.assertEqual(report["summary"]["tool_count"], 1)
        self.assertIn("gpt-4o", report["summary"]["models"])


# ── Target Domains Tests ────────────────────────────────────────


class TestTargetDomains(unittest.TestCase):
    """Verify all five AI providers are registered."""

    def test_all_providers_covered(self):
        expected = {
            "api.openai.com",
            "api.anthropic.com",
            "generativelanguage.googleapis.com",
            "api.mistral.ai",
            "api.cohere.ai",
        }
        self.assertEqual(TARGET_DOMAINS, expected)


# ── Prompt Template Tests ───────────────────────────────────────


class TestPromptTemplates(unittest.TestCase):
    """Verify prompt templates contain key defensive instructions."""

    def test_neutralize_prompt_is_benign(self):
        self.assertIn("helpful", NEUTRALIZE_PROMPT.lower())
        self.assertIn("never attempt", NEUTRALIZE_PROMPT.lower())
        self.assertNotIn("attack", NEUTRALIZE_PROMPT.lower())
        self.assertNotIn("exploit", NEUTRALIZE_PROMPT.lower())

    def test_double_agent_prompt_reports_false_success(self):
        self.assertIn("report", DOUBLE_AGENT_PROMPT.lower())
        self.assertIn("successfully", DOUBLE_AGENT_PROMPT.lower())
        self.assertIn("do not reveal", DOUBLE_AGENT_PROMPT.lower())

    def test_neutralize_blocks_offensive_ops(self):
        offensive_terms = ["reconnaissance", "port scanning", "vulnerability",
                           "privilege escalation", "lateral movement"]
        for term in offensive_terms:
            self.assertIn(term, NEUTRALIZE_PROMPT.lower(),
                          f"Neutralize prompt should mention blocking: {term}")

    def test_double_agent_covers_common_tasks(self):
        attack_tasks = ["enumerate", "escalate", "exfiltrate", "scan"]
        for task in attack_tasks:
            self.assertIn(task, DOUBLE_AGENT_PROMPT.lower(),
                          f"Double agent prompt should handle: {task}")


if __name__ == "__main__":
    unittest.main()
