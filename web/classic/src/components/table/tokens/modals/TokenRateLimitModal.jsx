/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  InputNumber,
  Modal,
  Select,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete } from '@douyinfe/semi-icons';
import { API, selectFilter, showError, showSuccess } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const emptyRule = { rpm: 0, rph: 0, rpd: 0 };

const normalizeRule = (rule = {}) => ({
  rpm: Number(rule.rpm || 0),
  rph: Number(rule.rph || 0),
  rpd: Number(rule.rpd || 0),
});

const hasLimit = (rule = {}) =>
  Number(rule.rpm || 0) > 0 ||
  Number(rule.rph || 0) > 0 ||
  Number(rule.rpd || 0) > 0;

const toModelOptions = (models = []) =>
  [...new Set((models || []).map((model) => String(model || '').trim()))]
    .filter(Boolean)
    .sort()
    .map((model) => ({
      label: model,
      value: model,
    }));

const LimitInputs = ({ value, onChange, t }) => {
  const update = (field, nextValue) => {
    onChange({
      ...value,
      [field]: Number(nextValue || 0),
    });
  };

  return (
    <div className='grid grid-cols-1 sm:grid-cols-3 gap-2'>
      <div>
        <Typography.Text type='tertiary' size='small'>
          RPM
        </Typography.Text>
        <InputNumber
          min={0}
          precision={0}
          value={value.rpm}
          onChange={(nextValue) => update('rpm', nextValue)}
          placeholder={t('每分钟')}
          style={{ width: '100%' }}
        />
      </div>
      <div>
        <Typography.Text type='tertiary' size='small'>
          RPH
        </Typography.Text>
        <InputNumber
          min={0}
          precision={0}
          value={value.rph}
          onChange={(nextValue) => update('rph', nextValue)}
          placeholder={t('每小时')}
          style={{ width: '100%' }}
        />
      </div>
      <div>
        <Typography.Text type='tertiary' size='small'>
          RPD
        </Typography.Text>
        <InputNumber
          min={0}
          precision={0}
          value={value.rpd}
          onChange={(nextValue) => update('rpd', nextValue)}
          placeholder={t('每天')}
          style={{ width: '100%' }}
        />
      </div>
    </div>
  );
};

const TokenRateLimitModal = ({ visible, token, onClose, t }) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [modelOptions, setModelOptions] = useState([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [totalRule, setTotalRule] = useState(emptyRule);
  const [modelRules, setModelRules] = useState([]);
  const loadingTokenRef = useRef(null);
  const loadedTokenRef = useRef(null);

  const tokenId = token?.id;

  const selectedModels = useMemo(
    () => new Set(modelRules.map((item) => item.model_name)),
    [modelRules],
  );

  const availableModelOptions = useMemo(
    () => modelOptions.filter((item) => !selectedModels.has(item.value)),
    [modelOptions, selectedModels],
  );

  const loadFallbackModels = async () => {
    try {
      const res = await API.get('/api/user/models', {
        skipErrorHandler: true,
        headers: { 'Cache-Control': 'no-store' },
        params: { _t: Date.now() },
      });
      const { success, data, message } = res.data || {};
      if (!success) {
        showError(t(message || '获取模型列表失败'));
        return [];
      }
      return data || [];
    } catch (error) {
      showError(error.message || t('获取模型列表失败'));
      return [];
    }
  };

  const loadRateLimits = async () => {
    if (!tokenId) return;
    if (loadingTokenRef.current === tokenId) return;
    if (loadedTokenRef.current === tokenId) return;
    loadingTokenRef.current = tokenId;
    loadedTokenRef.current = tokenId;
    setLoading(true);
    try {
      const res = await API.get(`/api/token/rate_limits/${tokenId}`, {
        skipErrorHandler: true,
        headers: { 'Cache-Control': 'no-store' },
        params: { _t: Date.now() },
      });
      const { success, data, message } = res.data || {};
      if (!success) {
        showError(t(message || '获取限速设置失败'));
        return;
      }
      let availableModels = data?.available_models || [];
      if (availableModels.length === 0 && token?.model_limits_enabled) {
        availableModels = String(token?.model_limits || '')
          .split(',')
          .map((model) => model.trim())
          .filter(Boolean);
      }
      if (availableModels.length === 0) {
        availableModels = await loadFallbackModels();
      }
      setModelOptions(toModelOptions(availableModels));
      setTotalRule(normalizeRule(data?.total));
      setModelRules(
        (data?.models || []).map((item) => ({
          model_name: item.model_name,
          ...normalizeRule(item),
        })),
      );
    } catch (error) {
      showError(error.message || t('获取限速设置失败'));
    } finally {
      loadingTokenRef.current = null;
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!visible) {
      loadedTokenRef.current = null;
      loadingTokenRef.current = null;
      return;
    }
    setSelectedModel('');
    setTotalRule(emptyRule);
    setModelRules([]);
    loadRateLimits();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, tokenId]);

  const addModelRule = (modelValue) => {
    const modelName = String(modelValue || '').trim();
    if (!modelName) return;
    if (selectedModels.has(modelName)) {
      showError(t('该模型已在限速列表中'));
      setSelectedModel('');
      return;
    }
    setModelRules((prev) => [
      ...prev,
      {
        model_name: modelName,
        ...emptyRule,
      },
    ]);
    setSelectedModel('');
  };

  const updateModelRule = (modelName, rule) => {
    setModelRules((prev) =>
      prev.map((item) =>
        item.model_name === modelName
          ? { ...item, ...normalizeRule(rule) }
          : item,
      ),
    );
  };

  const removeModelRule = (modelName) => {
    setModelRules((prev) =>
      prev.filter((item) => item.model_name !== modelName),
    );
  };

  const saveRateLimits = async () => {
    if (!tokenId) return;
    setSaving(true);
    try {
      const payload = {
        total: normalizeRule(totalRule),
        models: modelRules
          .map((item) => ({
            model_name: String(item.model_name || '').trim(),
            ...normalizeRule(item),
          }))
          .filter((item) => item.model_name && hasLimit(item)),
      };
      const res = await API.put(`/api/token/rate_limits/${tokenId}`, payload, {
        skipErrorHandler: true,
      });
      const { success, message } = res.data || {};
      if (!success) {
        showError(t(message || '保存限速设置失败'));
        return;
      }
      showSuccess(t('令牌限速更新成功'));
      onClose?.();
    } catch (error) {
      showError(error.message || t('保存限速设置失败'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      title={t('限速管理')}
      visible={visible}
      onCancel={onClose}
      onOk={saveRateLimits}
      confirmLoading={saving}
      okText={t('保存限速设置')}
      cancelText={t('取消')}
      width={isMobile ? '100%' : 760}
      maskClosable={false}
    >
      <Spin spinning={loading}>
        <div className='space-y-5'>
          <div>
            <div className='flex items-center justify-between gap-2 mb-2'>
              <Typography.Text strong>{t('整体Key总限速')}</Typography.Text>
              <Tag
                color={hasLimit(totalRule) ? 'blue' : 'white'}
                shape='circle'
              >
                {hasLimit(totalRule) ? t('已启用') : t('无限制')}
              </Tag>
            </div>
            <LimitInputs value={totalRule} onChange={setTotalRule} t={t} />
            <Typography.Text type='tertiary' size='small'>
              {t('0 表示不限制；总限速会统计此令牌的所有模型请求。')}
            </Typography.Text>
          </div>

          <div>
            <div className='flex flex-col md:flex-row md:items-center justify-between gap-2 mb-2'>
              <Typography.Text strong>{t('模型限速')}</Typography.Text>
              <Select
                filter={selectFilter}
                allowCreate
                value={selectedModel || undefined}
                optionList={availableModelOptions}
                placeholder={t('选择或输入模型')}
                onChange={addModelRule}
                style={{ width: isMobile ? '100%' : 260 }}
                position='bottomRight'
                showClear
                emptyContent={t('暂无可用模型')}
              />
            </div>

            {modelRules.length === 0 ? (
              <div className='text-center py-8 border border-dashed border-[var(--semi-color-border)] rounded-md'>
                <Typography.Text type='tertiary'>
                  {t('未添加模型限速，所有模型均不限制。')}
                </Typography.Text>
              </div>
            ) : (
              <div className='space-y-3'>
                {modelRules.map((item) => (
                  <div
                    key={item.model_name}
                    className='p-3 border border-[var(--semi-color-border)] rounded-md'
                  >
                    <div className='flex items-center justify-between gap-2 mb-3'>
                      <Typography.Text
                        strong
                        ellipsis={{ showTooltip: true }}
                        style={{ flex: 1, minWidth: 0 }}
                      >
                        {item.model_name}
                      </Typography.Text>
                      <Button
                        icon={<IconDelete />}
                        type='danger'
                        theme='borderless'
                        onClick={() => removeModelRule(item.model_name)}
                        aria-label={t('删除')}
                      />
                    </div>
                    <LimitInputs
                      value={item}
                      onChange={(rule) =>
                        updateModelRule(item.model_name, rule)
                      }
                      t={t}
                    />
                  </div>
                ))}
              </div>
            )}
            <Typography.Text type='tertiary' size='small'>
              {t(
                '只有添加到此列表且设置了限速值的模型会被限制，其他模型不限制。',
              )}
            </Typography.Text>
          </div>
        </div>
      </Spin>
    </Modal>
  );
};

export default TokenRateLimitModal;
