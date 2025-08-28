package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vera-byte/vgo-kit/i18n"
)

// TestI18nTranslation 测试国际化翻译功能
// 验证不同语言的翻译是否正确
func TestI18nTranslation(t *testing.T) {
	// 创建翻译器实例
	translator := i18n.NewTranslator(i18n.DefaultLanguage)

	// 加载翻译文件
	err := translator.LoadTranslations("../locales")
	assert.NoError(t, err, "加载翻译文件应该成功")

	// 测试英文翻译
	translator.SetLanguage(i18n.LanguageEnglish)
	message := translator.Translate("error.user.not_found", "testuser")
	assert.Contains(t, message, "User not found", "英文翻译应该包含正确内容")

	// 测试中文翻译
	translator.SetLanguage(i18n.LanguageChinese)
	message = translator.Translate("error.user.not_found", "testuser")
	assert.Contains(t, message, "用户不存在", "中文翻译应该包含正确内容")

	// 测试日文翻译
	translator.SetLanguage(i18n.LanguageJapanese)
	message = translator.Translate("error.user.not_found", "testuser")
	assert.Contains(t, message, "ユーザーが見つかりません", "日文翻译应该包含正确内容")

	// 测试韩文翻译
	translator.SetLanguage(i18n.LanguageKorean)
	message = translator.Translate("error.user.not_found", "testuser")
	assert.Contains(t, message, "사용자를 찾을 수 없습니다", "韩文翻译应该包含正确内容")
}

// TestLanguageDetection 测试语言检测功能
// 验证从Accept-Language头部解析语言是否正确
func TestLanguageDetection(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		expected       i18n.SupportedLanguage
	}{
		{
			name:           "英文",
			acceptLanguage: "en-US,en;q=0.9",
			expected:       i18n.LanguageEnglish,
		},
		{
			name:           "中文",
			acceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
			expected:       i18n.LanguageChinese,
		},
		{
			name:           "日文",
			acceptLanguage: "ja-JP,ja;q=0.9",
			expected:       i18n.LanguageJapanese,
		},
		{
			name:           "韩文",
			acceptLanguage: "ko-KR,ko;q=0.9",
			expected:       i18n.LanguageKorean,
		},
		{
			name:           "不支持的语言回退到默认",
			acceptLanguage: "fr-FR,fr;q=0.9",
			expected:       i18n.DefaultLanguage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang := i18n.ParseAcceptLanguage(tt.acceptLanguage)
			assert.Equal(t, tt.expected, lang, "语言检测结果应该正确")
		})
	}
}

// TestContextLanguage 测试上下文语言设置和获取
// 验证语言信息在上下文中的传递是否正确
func TestContextLanguage(t *testing.T) {
	ctx := context.Background()

	// 设置语言到上下文
	ctx = i18n.SetLanguageToContext(ctx, i18n.LanguageChinese)

	// 从上下文获取语言
	lang := i18n.GetLanguageFromContext(ctx)
	assert.Equal(t, i18n.LanguageChinese, lang, "从上下文获取的语言应该正确")

	// 测试默认语言
	emptyCtx := context.Background()
	defaultLang := i18n.GetLanguageFromContext(emptyCtx)
	assert.Equal(t, i18n.DefaultLanguage, defaultLang, "空上下文应该返回默认语言")
}