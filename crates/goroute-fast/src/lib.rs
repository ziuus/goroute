use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::sync::OnceLock;
use regex::Regex;
use serde::Serialize;

static PLEASANTRIES: OnceLock<Regex> = OnceLock::new();
static POLITE: OnceLock<Regex> = OnceLock::new();
static HEDGING: OnceLock<Regex> = OnceLock::new();
static VERBOSE: OnceLock<Regex> = OnceLock::new();
static CONNECTIVES: OnceLock<Regex> = OnceLock::new();
static IN_ORDER_TO: OnceLock<Regex> = OnceLock::new();
static AS_WELL_AS: OnceLock<Regex> = OnceLock::new();
static MULTI_NEWLINE: OnceLock<Regex> = OnceLock::new();
static MULTI_SPACE: OnceLock<Regex> = OnceLock::new();
static CODE_BLOCKS: OnceLock<Regex> = OnceLock::new();
static INLINE_CODE: OnceLock<Regex> = OnceLock::new();
static MD_LINKS: OnceLock<Regex> = OnceLock::new();

// PII & Secret Redaction Regexes
static PII_OPENAI_KEY: OnceLock<Regex> = OnceLock::new();
static PII_AWS_TOKEN: OnceLock<Regex> = OnceLock::new();
static PII_GOOGLE_TOKEN: OnceLock<Regex> = OnceLock::new();
static PII_JWT: OnceLock<Regex> = OnceLock::new();
static PII_PRIVKEY: OnceLock<Regex> = OnceLock::new();

fn init_regexes() {
    PLEASANTRIES.get_or_init(|| {
        Regex::new(r"(?i)\b(?:thank you so much|thank you|thanks a lot|thanks|no problem|you'?re welcome|i'?d be happy to|i would be glad to|glad to help|of course|certainly|cheers)\b[,.!?\s]*").unwrap()
    });
    POLITE.get_or_init(|| {
        Regex::new(r"(?i)\b(?:could you please|would you please|can you please|would you kindly|could you kindly|please|kindly|i would like you to|i need you to|i want you to|i would be (?:very )?grateful if you could|i would appreciate it if you could)\b\s*").unwrap()
    });
    HEDGING.get_or_init(|| {
        Regex::new(r"(?i)\b(?:it seems like|it appears that|i think that|i believe that|in my opinion|probably|possibly)\b\s*").unwrap()
    });
    VERBOSE.get_or_init(|| {
        Regex::new(r"(?i)\b(?:provide a detailed explanation of|give me a comprehensive explanation of|write an in-depth explanation of|give me an in-depth breakdown of|explain in detail|provide a step-by-step guide to)\b\s*").unwrap()
    });
    CONNECTIVES.get_or_init(|| {
        Regex::new(r"(?i)\b(?:due to the fact that|the reason is because|owing to the fact that)\b\s*").unwrap()
    });
    IN_ORDER_TO.get_or_init(|| {
        Regex::new(r"(?i)\b(?:in order to|for the purpose of)\b\s*").unwrap()
    });
    AS_WELL_AS.get_or_init(|| {
        Regex::new(r"(?i)\b(?:as well as)\b\s*").unwrap()
    });
    MULTI_NEWLINE.get_or_init(|| {
        Regex::new(r"\n{3,}").unwrap()
    });
    MULTI_SPACE.get_or_init(|| {
        Regex::new(r"[ \t]{2,}").unwrap()
    });
    CODE_BLOCKS.get_or_init(|| {
        Regex::new(r"(?s)```.*?```").unwrap()
    });
    INLINE_CODE.get_or_init(|| {
        Regex::new(r"`[^`\n]+`").unwrap()
    });
    MD_LINKS.get_or_init(|| {
        Regex::new(r"!?\[.*?\]\(.*?\)").unwrap()
    });

    PII_OPENAI_KEY.get_or_init(|| {
        Regex::new(r"sk-[A-Za-z0-9_\-]{24,}").unwrap()
    });
    PII_AWS_TOKEN.get_or_init(|| {
        Regex::new(r"aorAAAA[A-Za-z0-9_\-\.]{20,}").unwrap()
    });
    PII_GOOGLE_TOKEN.get_or_init(|| {
        Regex::new(r"1//0[A-Za-z0-9_\-]{40,}").unwrap()
    });
    PII_JWT.get_or_init(|| {
        Regex::new(r"eyJ[A-Za-z0-9_\-]{20,}\.eyJ[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{20,}").unwrap()
    });
    PII_PRIVKEY.get_or_init(|| {
        Regex::new(r"-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----").unwrap()
    });
}

#[derive(Serialize)]
pub struct ModelTarget {
    pub provider_id: String,
    pub target_model: String,
}

#[derive(Serialize)]
pub struct ModelResolution {
    pub resolved: bool,
    pub provider_id: String,
    pub target_model: String,
    pub family: String,
    pub fallbacks: Vec<ModelTarget>,
}

#[no_mangle]
pub extern "C" fn goroute_rust_version() -> *const c_char {
    static VERSION: &[u8] = b"GoRoute Rust Accelerator v0.3.0 (SIMD Routing + Guardrails + Caveman)\0";
    VERSION.as_ptr() as *const c_char
}

#[no_mangle]
pub extern "C" fn goroute_rust_resolve_model(model_ptr: *const c_char, providers_ptr: *const c_char) -> *mut c_char {
    if model_ptr.is_null() {
        return std::ptr::null_mut();
    }

    let model = match unsafe { CStr::from_ptr(model_ptr) }.to_str() {
        Ok(s) => s.trim().to_lowercase(),
        Err(_) => return std::ptr::null_mut(),
    };

    let providers_csv = if !providers_ptr.is_null() {
        unsafe { CStr::from_ptr(providers_ptr) }.to_str().unwrap_or("")
    } else {
        ""
    };

    let active_provs: Vec<&str> = providers_csv
        .split(',')
        .map(|s| s.trim())
        .filter(|s| !s.is_empty())
        .collect();

    let has = |p: &str| active_provs.contains(&p);

    // 1. Direct provider/model syntax e.g. "antigravity/gemini-2.5-pro"
    if model.contains('/') {
        let parts: Vec<&str> = model.splitn(2, '/').collect();
        let p_id = parts[0];
        let m_name = parts[1];
        let res = ModelResolution {
            resolved: true,
            provider_id: p_id.to_string(),
            target_model: m_name.to_string(),
            family: "custom".to_string(),
            fallbacks: vec![],
        };
        let json_str = serde_json::to_string(&res).unwrap_or_default();
        return CString::new(json_str).unwrap().into_raw();
    }

    // 2. Intelligent Family Routing
    let mut primary_prov = "";
    let mut primary_target = "";
    let mut family = "unknown";
    let mut fallbacks = Vec::new();

    // Claude Sonnet 3.7 / 4.6
    if model.contains("3-7") || model.contains("3.7") || model.contains("sonnet-3-7") || model.contains("4-6") {
        family = "claude";
        if has("antigravity") {
            primary_prov = "antigravity";
            primary_target = "claude-sonnet-4-6";
            if has("kiro") {
                fallbacks.push(ModelTarget {
                    provider_id: "kiro".to_string(),
                    target_model: "claude-sonnet-4.5".to_string(),
                });
            }
        } else if has("kiro") {
            primary_prov = "kiro";
            primary_target = "claude-sonnet-4.5";
        }
    }
    // Claude Sonnet 3.5 / 4.5
    else if model.contains("sonnet") || model.contains("3-5-sonnet") || model.contains("3.5-sonnet") {
        family = "claude";
        if has("kiro") {
            primary_prov = "kiro";
            primary_target = "claude-sonnet-4.5";
            if has("antigravity") {
                fallbacks.push(ModelTarget {
                    provider_id: "antigravity".to_string(),
                    target_model: "claude-sonnet-4-6".to_string(),
                });
            }
        } else if has("antigravity") {
            primary_prov = "antigravity";
            primary_target = "claude-sonnet-4-6";
        }
    }
    // Claude Haiku
    else if model.contains("haiku") {
        family = "claude";
        if has("kiro") {
            primary_prov = "kiro";
            primary_target = "claude-haiku-4.5";
        }
    }
    // Claude Opus
    else if model.contains("opus") {
        family = "claude";
        if has("antigravity") {
            primary_prov = "antigravity";
            primary_target = "claude-opus-4-6-thinking";
            if has("kiro") {
                fallbacks.push(ModelTarget {
                    provider_id: "kiro".to_string(),
                    target_model: "claude-sonnet-4.5".to_string(),
                });
            }
        } else if has("kiro") {
            primary_prov = "kiro";
            primary_target = "claude-sonnet-4.5";
        }
    }
    // Generic Claude
    else if model.contains("claude") {
        family = "claude";
        if has("antigravity") {
            primary_prov = "antigravity";
            primary_target = "claude-sonnet-4-6";
            if has("kiro") {
                fallbacks.push(ModelTarget {
                    provider_id: "kiro".to_string(),
                    target_model: "claude-sonnet-4.5".to_string(),
                });
            }
        } else if has("kiro") {
            primary_prov = "kiro";
            primary_target = "claude-sonnet-4.5";
        }
    }
    // OpenAI o1 / o3 reasoning
    else if model.starts_with("o1") || model.starts_with("o3") || model.contains("reasoning") {
        family = "openai";
        if has("codex") {
            primary_prov = "codex";
            primary_target = if model.contains("mini") { "o3-mini" } else { "o3-mini" };
            if has("agentrouter") {
                fallbacks.push(ModelTarget {
                    provider_id: "agentrouter".to_string(),
                    target_model: "gpt-6-astra".to_string(),
                });
            }
        } else if has("agentrouter") {
            primary_prov = "agentrouter";
            primary_target = "gpt-6-astra";
        }
    }
    // OpenAI GPT-4 / GPT-4o
    else if model.contains("gpt-4") || model.contains("4o") {
        family = "openai";
        if has("codex") {
            primary_prov = "codex";
            primary_target = "gpt-4o";
            if has("openai-compatible") {
                fallbacks.push(ModelTarget {
                    provider_id: "openai-compatible".to_string(),
                    target_model: "gemini-3.1-pro".to_string(),
                });
            }
        } else if has("openai-compatible") {
            primary_prov = "openai-compatible";
            primary_target = "gemini-3.1-pro";
        }
    }
    // AgentRouter frontier: gpt-5, gpt-6, astra, sol
    else if model.contains("gpt-5") || model.contains("gpt-6") || model.contains("astra") || model.contains("sol") {
        family = "agentrouter";
        if has("agentrouter") {
            primary_prov = "agentrouter";
            primary_target = if model.contains("sol") { "gpt-5.6-sol" } else { "gpt-6-astra" };
        }
    }
    // Gemini Family
    else if model.contains("gemini") || model.contains("flash") {
        family = "gemini";
        if has("antigravity") {
            primary_prov = "antigravity";
            primary_target = if model.contains("pro") { "gemini-3.1-pro-high" } else { "gemini-3.8-flash-high" };
            if has("openai-compatible") {
                fallbacks.push(ModelTarget {
                    provider_id: "openai-compatible".to_string(),
                    target_model: "gemini-3.1-pro".to_string(),
                });
            }
        } else if has("openai-compatible") {
            primary_prov = "openai-compatible";
            primary_target = "gemini-3.1-pro";
        }
    }
    // DeepSeek Family
    else if model.contains("deepseek") {
        family = "deepseek";
        if has("kiro") {
            primary_prov = "kiro";
            primary_target = "deepseek-3.2";
            if has("openai-compatible") {
                fallbacks.push(ModelTarget {
                    provider_id: "openai-compatible".to_string(),
                    target_model: "deepseek-v4-pro".to_string(),
                });
            }
        } else if has("openai-compatible") {
            primary_prov = "openai-compatible";
            primary_target = "deepseek-v4-pro";
        }
    }

    let resolved = !primary_prov.is_empty();
    let res = ModelResolution {
        resolved,
        provider_id: primary_prov.to_string(),
        target_model: primary_target.to_string(),
        family: family.to_string(),
        fallbacks,
    };

    let json_str = serde_json::to_string(&res).unwrap_or_default();
    CString::new(json_str).unwrap().into_raw()
}

#[no_mangle]
pub extern "C" fn goroute_rust_sanitize_pii(input: *const c_char) -> *mut c_char {
    if input.is_null() {
        return std::ptr::null_mut();
    }
    let c_str = unsafe { CStr::from_ptr(input) };
    let text = match c_str.to_str() {
        Ok(s) => s,
        Err(_) => return std::ptr::null_mut(),
    };
    if text.is_empty() {
        return CString::new("").unwrap().into_raw();
    }

    init_regexes();

    let mut redacted = text.to_string();
    redacted = PII_OPENAI_KEY.get().unwrap().replace_all(&redacted, "[REDACTED_API_KEY]").into_owned();
    redacted = PII_AWS_TOKEN.get().unwrap().replace_all(&redacted, "[REDACTED_AWS_TOKEN]").into_owned();
    redacted = PII_GOOGLE_TOKEN.get().unwrap().replace_all(&redacted, "[REDACTED_GOOGLE_TOKEN]").into_owned();
    redacted = PII_JWT.get().unwrap().replace_all(&redacted, "[REDACTED_JWT]").into_owned();
    redacted = PII_PRIVKEY.get().unwrap().replace_all(&redacted, "[REDACTED_PRIVATE_KEY]").into_owned();

    match CString::new(redacted) {
        Ok(c_out) => c_out.into_raw(),
        Err(_) => std::ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn goroute_rust_compress(input: *const c_char, level: i32) -> *mut c_char {
    if input.is_null() {
        return std::ptr::null_mut();
    }

    let c_str = unsafe { CStr::from_ptr(input) };
    let input_str = match c_str.to_str() {
        Ok(s) => s,
        Err(_) => return std::ptr::null_mut(),
    };

    if input_str.is_empty() {
        return CString::new("").unwrap().into_raw();
    }

    init_regexes();

    // 1. Extract & protect code blocks, inline code, and links
    let mut protected = Vec::new();

    let text1 = CODE_BLOCKS.get().unwrap().replace_all(input_str, |caps: &regex::Captures| {
        let idx = protected.len();
        protected.push(caps[0].to_string());
        format!("___GR_PROT_{}___", idx)
    });

    let text2 = INLINE_CODE.get().unwrap().replace_all(&text1, |caps: &regex::Captures| {
        let idx = protected.len();
        protected.push(caps[0].to_string());
        format!("___GR_PROT_{}___", idx)
    });

    let text3 = MD_LINKS.get().unwrap().replace_all(&text2, |caps: &regex::Captures| {
        let idx = protected.len();
        protected.push(caps[0].to_string());
        format!("___GR_PROT_{}___", idx)
    });

    let mut prose = text3.into_owned();

    // 2. Multi-tier prompt compression
    if level >= 1 {
        prose = PLEASANTRIES.get().unwrap().replace_all(&prose, "").into_owned();
        prose = POLITE.get().unwrap().replace_all(&prose, "").into_owned();
        prose = MULTI_NEWLINE.get().unwrap().replace_all(&prose, "\n\n").into_owned();
        prose = MULTI_SPACE.get().unwrap().replace_all(&prose, " ").into_owned();
    }

    if level >= 2 {
        prose = HEDGING.get().unwrap().replace_all(&prose, "").into_owned();
        prose = VERBOSE.get().unwrap().replace_all(&prose, "").into_owned();
    }

    if level >= 3 {
        prose = CONNECTIVES.get().unwrap().replace_all(&prose, "because ").into_owned();
        prose = IN_ORDER_TO.get().unwrap().replace_all(&prose, "to ").into_owned();
        prose = AS_WELL_AS.get().unwrap().replace_all(&prose, "and ").into_owned();
    }

    // 3. Restore protected literals in reverse order
    for (i, code) in protected.iter().enumerate().rev() {
        let placeholder = format!("___GR_PROT_{}___", i);
        prose = prose.replace(&placeholder, code);
    }

    match CString::new(prose.trim()) {
        Ok(c_out) => c_out.into_raw(),
        Err(_) => std::ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn goroute_rust_count_tokens(input: *const c_char) -> i32 {
    if input.is_null() {
        return 0;
    }
    let c_str = unsafe { CStr::from_ptr(input) };
    let text = match c_str.to_str() {
        Ok(s) => s,
        Err(_) => return 0,
    };
    if text.is_empty() {
        return 0;
    }

    let char_count = text.chars().count();
    let word_count = text.split_whitespace().count();
    let estimated = (char_count as f64 / 3.75).round() as i32;
    if estimated < word_count as i32 {
        word_count as i32
    } else {
        estimated
    }
}

#[no_mangle]
pub extern "C" fn goroute_rust_free_string(ptr: *mut c_char) {
    if !ptr.is_null() {
        unsafe {
            let _ = CString::from_raw(ptr);
        }
    }
}
