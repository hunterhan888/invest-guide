import { useEffect, useState } from 'react';
import { App as AntdApp, Button, Checkbox, Form, Input } from 'antd';
import { CompassOutlined, LockOutlined, MailOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { login as apiLogin } from '@/api/auth/auth';
import { useAuthStore } from '@/stores/authStore';
import type { LoginRequest } from '@/api/auth/types';

type LoginFormValues = LoginRequest & { remember?: boolean };

export default function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = AntdApp.useApp();
  const token = useAuthStore((s) => s.token);
  const loginStore = useAuthStore((s) => s.login);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm<LoginFormValues>();

  useEffect(() => {
    if (token) navigate('/', { replace: true });
  }, [token, navigate]);

  async function onLogin(values: LoginFormValues) {
    setLoading(true);
    try {
      const res = await apiLogin({ email: values.email, password: values.password });
      loginStore({ token: res.token, user: res.user, remember: values.remember !== false });
      navigate('/', { replace: true });
    } catch (e: unknown) {
      const code = (e as { code?: string })?.code;
      message.error(code === 'UNAUTHORIZED' ? t('auth.error.invalid') : t('error.generic'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-layout px-4">
      <div className="w-full max-w-[360px]">
        <div className="mb-8 text-center">
          <div
            className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl"
            style={{ background: 'linear-gradient(135deg, #4096ff, #1677ff)' }}
          >
            <CompassOutlined style={{ fontSize: 26, color: '#fff' }} />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-fg">{t('auth.login.title')}</h1>
          <p className="mt-1 text-fg-secondary">{t('auth.login.subtitle')}</p>
        </div>
        <Form form={form} layout="vertical" onFinish={onLogin} initialValues={{ remember: true }}>
          <Form.Item name="email" rules={[{ required: true }, { type: 'email' }]}>
            <Input
              prefix={<MailOutlined className="text-fg-tertiary" />}
              placeholder={t('auth.field.email')}
              size="large"
            />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true }, { min: 8 }, { max: 72 }]}>
            <Input.Password
              prefix={<LockOutlined className="text-fg-tertiary" />}
              placeholder={t('auth.field.password')}
              size="large"
            />
          </Form.Item>
          <Form.Item className="mb-4">
            <Form.Item name="remember" valuePropName="checked" noStyle>
              <Checkbox>{t('auth.login.remember')}</Checkbox>
            </Form.Item>
          </Form.Item>
          <Button type="primary" htmlType="submit" block size="large" loading={loading}>
            {t('auth.login.submit')}
          </Button>
        </Form>
        <div className="mt-4 text-center">
          <Button type="link" onClick={() => navigate('/register')}>
            {t('auth.login.toRegister')}
          </Button>
        </div>
      </div>
    </div>
  );
}
