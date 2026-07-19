import { useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Animated,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { ApiError } from '@/core/api';
import {
  createKnowledgeBase,
  validateKnowledgeBaseInput,
  type KnowledgeBase,
  type KnowledgeBaseInputErrors,
} from '@/core/knowledge';
import { type Palette } from '@/theme/palette';

export function CreateKnowledgeSheet({
  palette,
  onClose,
  onCreated,
  onSessionExpired,
}: {
  palette: Palette;
  onClose: () => void;
  onCreated: (knowledgeBase: KnowledgeBase) => void;
  onSessionExpired: () => Promise<void>;
}) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [errors, setErrors] = useState<KnowledgeBaseInputErrors>({});
  const [formError, setFormError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [progress] = useState(() => new Animated.Value(0));
  const descriptionRef = useRef<TextInput>(null);
  const closingRef = useRef(false);
  const submittingRef = useRef(false);
  const insets = useSafeAreaInsets();

  useEffect(() => {
    progress.setValue(0);
    const frame = requestAnimationFrame(() => {
      Animated.spring(progress, {
        toValue: 1,
        damping: 25,
        stiffness: 270,
        mass: 0.85,
        useNativeDriver: true,
      }).start();
    });
    return () => {
      cancelAnimationFrame(frame);
      progress.stopAnimation();
    };
  }, [progress]);

  function animateClosed(afterClose?: () => void) {
    if (closingRef.current) {
      return;
    }
    closingRef.current = true;
    Animated.timing(progress, {
      toValue: 0,
      duration: 190,
      useNativeDriver: true,
    }).start(({ finished }) => {
      closingRef.current = false;
      if (finished) {
        onClose();
        afterClose?.();
      }
    });
  }

  function close() {
    if (!submittingRef.current) {
      animateClosed();
    }
  }

  async function submit() {
    if (submittingRef.current || closingRef.current) {
      return;
    }
    const input = { name, description };
    const nextErrors = validateKnowledgeBaseInput(input);
    setErrors(nextErrors);
    setFormError('');
    if (Object.keys(nextErrors).length > 0) {
      return;
    }

    submittingRef.current = true;
    setSubmitting(true);
    let createdSuccessfully = false;
    try {
      const created = await createKnowledgeBase(input);
      createdSuccessfully = true;
      animateClosed(() => onCreated(created));
    } catch (caught) {
      if (caught instanceof ApiError) {
        const serverErrors: KnowledgeBaseInputErrors = {};
        for (const item of caught.fieldErrors) {
          if (item.field === 'name' || item.field === 'description') {
            serverErrors[item.field] = item.message;
          }
        }
        setErrors(serverErrors);
        setFormError(caught.message);
        if (caught.status === 401) {
          await onSessionExpired();
        }
      } else {
        setFormError('创建知识库失败，请稍后重试。');
      }
    } finally {
      if (!createdSuccessfully) {
        submittingRef.current = false;
        setSubmitting(false);
      }
    }
  }

  const translateY = progress.interpolate({ inputRange: [0, 1], outputRange: [420, 0] });

  return (
    <Modal
      transparent
      presentationStyle="overFullScreen"
      statusBarTranslucent
      animationType="none"
      onRequestClose={close}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        style={styles.modalRoot}>
        <View style={[StyleSheet.absoluteFill, { backgroundColor: palette.scrim }]}>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="关闭创建知识库面板"
            disabled={submitting}
            onPress={close}
            testID="knowledge-create-dismiss"
            style={StyleSheet.absoluteFill}
          />
        </View>
        <Animated.View
          style={[
            styles.sheet,
            {
              backgroundColor: palette.surface,
              borderColor: palette.border,
              paddingBottom: Math.max(insets.bottom, 24),
              transform: [{ translateY }],
            },
          ]}>
          <View style={[styles.grabber, { backgroundColor: palette.borderStrong }]} />
          <View style={styles.sheetHeader}>
            <Pressable
              accessibilityRole="button"
              accessibilityLabel="取消创建知识库"
              disabled={submitting}
              hitSlop={8}
              onPress={close}
              testID="knowledge-create-cancel"
              style={({ pressed }) => pressed && styles.pressed}>
              <Text style={[styles.sheetAction, { color: palette.accent }]}>取消</Text>
            </Pressable>
            <Text style={[styles.sheetTitle, { color: palette.text }]}>新建知识库</Text>
            <Pressable
              accessibilityRole="button"
              accessibilityLabel="创建知识库"
              accessibilityState={{ disabled: submitting, busy: submitting }}
              disabled={submitting}
              hitSlop={8}
              onPress={() => void submit()}
              testID="knowledge-create-submit"
              style={({ pressed }) => pressed && !submitting && styles.pressed}>
              <View style={styles.submitContent}>
                {submitting ? <ActivityIndicator size="small" color={palette.accent} /> : null}
                <Text style={[styles.sheetAction, styles.saveAction, { color: palette.accent }]}>创建</Text>
              </View>
            </Pressable>
          </View>
          <ScrollView
            keyboardShouldPersistTaps="handled"
            contentContainerStyle={styles.sheetForm}>
            <View style={styles.field}>
              <Text style={[styles.label, { color: palette.text }]}>名称</Text>
              <TextInput
                accessibilityLabel="知识库名称"
                autoCapitalize="sentences"
                autoComplete="off"
                maxLength={128}
                onChangeText={(value) => {
                  setName(value);
                  if (errors.name) {
                    setErrors((current) => ({ ...current, name: undefined }));
                  }
                }}
                onSubmitEditing={() => descriptionRef.current?.focus()}
                placeholder="例如：产品资料"
                placeholderTextColor={palette.textMuted}
                returnKeyType="next"
                selectionColor={palette.accent}
                style={[
                  styles.input,
                  {
                    color: palette.text,
                    backgroundColor: palette.input,
                    borderColor: errors.name ? palette.danger : palette.borderStrong,
                  },
                ]}
                testID="knowledge-name-input"
                value={name}
              />
              {errors.name ? <Text style={[styles.fieldError, { color: palette.danger }]}>{errors.name}</Text> : null}
            </View>
            <View style={styles.field}>
              <Text style={[styles.label, { color: palette.text }]}>描述</Text>
              <TextInput
                ref={descriptionRef}
                accessibilityLabel="知识库描述"
                maxLength={512}
                multiline
                onChangeText={(value) => {
                  setDescription(value);
                  if (errors.description) {
                    setErrors((current) => ({ ...current, description: undefined }));
                  }
                }}
                placeholder="可选，说明这里会收纳什么内容"
                placeholderTextColor={palette.textMuted}
                selectionColor={palette.accent}
                style={[
                  styles.input,
                  styles.descriptionInput,
                  {
                    color: palette.text,
                    backgroundColor: palette.input,
                    borderColor: errors.description ? palette.danger : palette.borderStrong,
                  },
                ]}
                testID="knowledge-description-input"
                textAlignVertical="top"
                value={description}
              />
              <Text style={[styles.counter, { color: palette.textMuted }]}>{description.length}/512</Text>
              {errors.description ? (
                <Text style={[styles.fieldError, { color: palette.danger }]}>{errors.description}</Text>
              ) : null}
            </View>
            {formError ? (
              <Text
                accessibilityLiveRegion="polite"
                style={[styles.formError, { color: palette.danger }]}
                testID="knowledge-create-error">
                {formError}
              </Text>
            ) : null}
          </ScrollView>
        </Animated.View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  modalRoot: { flex: 1, justifyContent: 'flex-end' },
  sheet: {
    minHeight: 390,
    maxHeight: '82%',
    borderTopLeftRadius: 22,
    borderTopRightRadius: 22,
    borderWidth: StyleSheet.hairlineWidth,
    boxShadow: '0 -8px 24px rgba(8, 31, 35, 0.14)',
  },
  grabber: { alignSelf: 'center', width: 36, height: 5, marginTop: 5, borderRadius: 3 },
  sheetHeader: {
    height: 57,
    paddingHorizontal: 19,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  sheetTitle: {
    position: 'absolute',
    left: 90,
    right: 90,
    textAlign: 'center',
    fontSize: 16,
    lineHeight: 21,
    fontWeight: '600',
  },
  sheetAction: { minWidth: 44, fontSize: 15, lineHeight: 20 },
  saveAction: { minWidth: 0, textAlign: 'right', fontWeight: '600' },
  submitContent: { minWidth: 44, flexDirection: 'row', alignItems: 'center', justifyContent: 'flex-end', gap: 6 },
  sheetForm: { paddingHorizontal: 19, paddingBottom: 10, gap: 16 },
  field: { gap: 6 },
  label: { fontSize: 13, lineHeight: 17, fontWeight: '600' },
  input: { height: 48, borderRadius: 12, borderWidth: 1, paddingHorizontal: 12, fontSize: 16 },
  descriptionInput: { height: 104, paddingTop: 12, paddingBottom: 12 },
  counter: { marginTop: -2, fontSize: 11, lineHeight: 14, textAlign: 'right' },
  fieldError: { fontSize: 11, lineHeight: 14 },
  formError: { fontSize: 12, lineHeight: 17 },
  pressed: { opacity: 0.55, transform: [{ scale: 0.97 }] },
});
