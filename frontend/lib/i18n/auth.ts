import type { Locale } from "./locales"

type AuthCopy = {
  common: {
    brand: string
    email: string
    password: string
    confirmPassword: string
    newPassword: string
    captchaRequired: string
    networkError: string
  }
  login: {
    subtitle: string
    forgot: string
    submit: string
    loading: string
    registerPrompt: string
    registerLink: string
  }
  register: {
    subtitle: string
    disabled: string
    emailRequired: string
    passwordPlaceholder: string
    confirmPlaceholder: string
    passwordMismatch: string
    passwordWeak: string
    codeRequired: string
    codeLabel: string
    codePlaceholder: string
    sendCode: string
    sending: string
    resend: string
    sendSuccess: string
    sendError: string
    submit: string
    loading: string
    submitError: string
    loginPrompt: string
    loginLink: string
  }
  forgot: {
    subtitle: string
    codeLabel: string
    codePlaceholder: string
    sendCode: string
    sending: string
    resend: string
    sendSuccess: string
    sendError: string
    codeRequired: string
    passwordMismatch: string
    passwordWeak: string
    submit: string
    loading: string
    done: string
    success: string
    submitError: string
    backPrompt: string
    backLink: string
  }
}

const zhCN: AuthCopy = {
  common: {
    brand: "Zboard",
    email: "邮箱地址",
    password: "密码",
    confirmPassword: "确认密码",
    newPassword: "新密码",
    captchaRequired: "请先完成安全验证",
    networkError: "网络错误，请稍后重试",
  },
  login: {
    subtitle: "进入多端协同加速控制中心",
    forgot: "忘记密码？",
    submit: "登录",
    loading: "正在登录...",
    registerPrompt: "还没有账户？",
    registerLink: "立即注册",
  },
  register: {
    subtitle: "创建新账户，开启云同步加速",
    disabled: "当前站点已关闭自主注册，请联系管理员创建账户。",
    emailRequired: "请先填写邮箱地址",
    passwordPlaceholder: "请输入至少 6 位密码",
    confirmPlaceholder: "请再次输入密码",
    passwordMismatch: "两次输入的密码不一致",
    passwordWeak: "密码至少需要 6 位",
    codeRequired: "请填写邮箱验证码",
    codeLabel: "邮箱验证码",
    codePlaceholder: "6 位数字",
    sendCode: "获取验证码",
    sending: "发送中...",
    resend: "重新发送",
    sendSuccess: "验证码已发送，请检查邮箱和垃圾箱。",
    sendError: "验证码发送失败，请重试",
    submit: "创建账户",
    loading: "正在创建账户...",
    submitError: "注册失败，请稍后重试",
    loginPrompt: "已有账户？",
    loginLink: "立即登录",
  },
  forgot: {
    subtitle: "重置账户密码，恢复访问权限",
    codeLabel: "邮箱验证码",
    codePlaceholder: "6 位数字",
    sendCode: "获取验证码",
    sending: "发送中...",
    resend: "重新发送",
    sendSuccess: "重置验证码已发送，请检查邮箱。",
    sendError: "验证码发送失败，请重试",
    codeRequired: "请填写邮箱验证码",
    passwordMismatch: "两次输入的密码不一致",
    passwordWeak: "密码至少需要 6 位",
    submit: "重置密码",
    loading: "正在重置...",
    done: "重置成功",
    success: "密码已重置，正在跳转到登录页...",
    submitError: "重置失败，请重试",
    backPrompt: "想起密码了？",
    backLink: "返回登录",
  },
}

const en: AuthCopy = {
  common: {
    brand: "Zboard",
    email: "Email",
    password: "Password",
    confirmPassword: "Confirm password",
    newPassword: "New password",
    captchaRequired: "Please complete the security check first.",
    networkError: "Network error. Please try again later.",
  },
  login: {
    subtitle: "Sign in to your multi-device control center",
    forgot: "Forgot password?",
    submit: "Sign in",
    loading: "Signing in...",
    registerPrompt: "Do not have an account?",
    registerLink: "Create one",
  },
  register: {
    subtitle: "Create an account and start using Zboard",
    disabled: "Self registration is currently disabled. Please contact the administrator.",
    emailRequired: "Please enter your email first.",
    passwordPlaceholder: "Enter at least 6 characters",
    confirmPlaceholder: "Enter the password again",
    passwordMismatch: "The two passwords do not match.",
    passwordWeak: "Password must be at least 6 characters.",
    codeRequired: "Please enter the email verification code.",
    codeLabel: "Email code",
    codePlaceholder: "6 digits",
    sendCode: "Send code",
    sending: "Sending...",
    resend: "Resend",
    sendSuccess: "Verification code sent. Check your inbox and spam folder.",
    sendError: "Failed to send the code. Please try again.",
    submit: "Create account",
    loading: "Creating account...",
    submitError: "Registration failed. Please try again later.",
    loginPrompt: "Already have an account?",
    loginLink: "Sign in",
  },
  forgot: {
    subtitle: "Reset your password and restore access",
    codeLabel: "Email code",
    codePlaceholder: "6 digits",
    sendCode: "Send code",
    sending: "Sending...",
    resend: "Resend",
    sendSuccess: "Reset code sent. Please check your email.",
    sendError: "Failed to send the code. Please try again.",
    codeRequired: "Please enter the email verification code.",
    passwordMismatch: "The two passwords do not match.",
    passwordWeak: "Password must be at least 6 characters.",
    submit: "Reset password",
    loading: "Resetting...",
    done: "Password reset",
    success: "Password reset. Redirecting to sign in...",
    submitError: "Reset failed. Please try again.",
    backPrompt: "Remembered your password?",
    backLink: "Back to sign in",
  },
}

export function authCopy(locale: Locale): AuthCopy {
  if (locale === "zh-CN" || locale === "zh-TW") return zhCN
  return en
}
