"use client";

import { useState, useEffect } from "react";

interface BrandingSettings {
  product_name: string;
  logo_url: string;
  favicon_url: string;
  primary_color: string;
  secondary_color: string;
  dashboard_domain: string;
  custom_email_from: string;
  support_email: string;
  support_url: string;
  hide_powered_by: boolean;
  terms_url: string;
  privacy_url: string;
}

export default function AdminBrandingPage() {
  const [branding, setBranding] = useState<BrandingSettings>({
    product_name: "FortressWAF",
    logo_url: "",
    favicon_url: "",
    primary_color: "#6366f1",
    secondary_color: "#8b5cf6",
    dashboard_domain: "",
    custom_email_from: "",
    support_email: "",
    support_url: "",
    hide_powered_by: false,
    terms_url: "",
    privacy_url: "",
  });
  const [previewMode, setPreviewMode] = useState<"light" | "dark">("light");
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [logoPreview, setLogoPreview] = useState<string | null>(null);
  const [faviconPreview, setFaviconPreview] = useState<string | null>(null);

  useEffect(() => {
    fetchBranding();
  }, []);

  const fetchBranding = async () => {
    try {
      const response = await fetch("/api/admin/branding");
      if (response.ok) {
        const data = await response.json();
        setBranding(data);
        if (data.logo_url) setLogoPreview(data.logo_url);
        if (data.favicon_url) setFaviconPreview(data.favicon_url);
      }
    } catch (error) {
      console.error("Failed to fetch branding:", error);
    }
  };

  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onloadend = () => {
        const result = reader.result as string;
        setLogoPreview(result);
        setBranding({ ...branding, logo_url: result });
      };
      reader.readAsDataURL(file);
    }
  };

  const handleFaviconUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onloadend = () => {
        const result = reader.result as string;
        setFaviconPreview(result);
        setBranding({ ...branding, favicon_url: result });
      };
      reader.readAsDataURL(file);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setSaved(false);
    try {
      const response = await fetch("/api/admin/branding", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(branding),
      });
      if (response.ok) {
        setSaved(true);
        setTimeout(() => setSaved(false), 3000);
      }
    } catch (error) {
      console.error("Failed to save branding:", error);
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setBranding({
      product_name: "FortressWAF",
      logo_url: "",
      favicon_url: "",
      primary_color: "#6366f1",
      secondary_color: "#8b5cf6",
      dashboard_domain: "",
      custom_email_from: "",
      support_email: "",
      support_url: "",
      hide_powered_by: false,
      terms_url: "",
      privacy_url: "",
    });
    setLogoPreview(null);
    setFaviconPreview(null);
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
            White-Label Branding
          </h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            Customize the dashboard appearance and branding for your organization
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm">
              <div className="p-6 border-b border-gray-200 dark:border-gray-700">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Product Identity
                </h2>
              </div>
              <div className="p-6 space-y-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Product Name
                  </label>
                  <input
                    type="text"
                    value={branding.product_name}
                    onChange={(e) => setBranding({ ...branding, product_name: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                    placeholder="FortressWAF"
                  />
                  <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    This name will appear throughout the dashboard and emails
                  </p>
                </div>

                <div className="grid grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Logo
                    </label>
                    <div className="flex items-start gap-4">
                      <div className="flex-shrink-0">
                        {logoPreview ? (
                          <div className="relative">
                            <img
                              src={logoPreview}
                              alt="Logo preview"
                              className="w-24 h-24 object-contain rounded-lg border border-gray-300 dark:border-gray-600 bg-white"
                            />
                            <button
                              onClick={() => {
                                setLogoPreview(null);
                                setBranding({ ...branding, logo_url: "" });
                              }}
                              className="absolute -top-2 -right-2 p-1 bg-red-500 text-white rounded-full hover:bg-red-600"
                            >
                              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                              </svg>
                            </button>
                          </div>
                        ) : (
                          <div className="w-24 h-24 rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-600 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
                            <svg className="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                            </svg>
                          </div>
                        )}
                      </div>
                      <div className="flex-1">
                        <input
                          type="file"
                          id="logo-upload"
                          accept="image/*"
                          onChange={handleLogoUpload}
                          className="hidden"
                        />
                        <label
                          htmlFor="logo-upload"
                          className="inline-flex items-center px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors"
                        >
                          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                          </svg>
                          Upload Logo
                        </label>
                        <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                          PNG, JPG up to 2MB. Recommended: 200x60px
                        </p>
                      </div>
                    </div>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Favicon
                    </label>
                    <div className="flex items-start gap-4">
                      <div className="flex-shrink-0">
                        {faviconPreview ? (
                          <div className="relative">
                            <img
                              src={faviconPreview}
                              alt="Favicon preview"
                              className="w-16 h-16 object-contain rounded-lg border border-gray-300 dark:border-gray-600 bg-white"
                            />
                            <button
                              onClick={() => {
                                setFaviconPreview(null);
                                setBranding({ ...branding, favicon_url: "" });
                              }}
                              className="absolute -top-2 -right-2 p-1 bg-red-500 text-white rounded-full hover:bg-red-600"
                            >
                              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                              </svg>
                            </button>
                          </div>
                        ) : (
                          <div className="w-16 h-16 rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-600 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
                            <svg className="w-6 h-6 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                            </svg>
                          </div>
                        )}
                      </div>
                      <div className="flex-1">
                        <input
                          type="file"
                          id="favicon-upload"
                          accept="image/*"
                          onChange={handleFaviconUpload}
                          className="hidden"
                        />
                        <label
                          htmlFor="favicon-upload"
                          className="inline-flex items-center px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors"
                        >
                          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                          </svg>
                          Upload Favicon
                        </label>
                        <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                          PNG, ICO up to 512KB. Recommended: 32x32px
                        </p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm">
              <div className="p-6 border-b border-gray-200 dark:border-gray-700">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Color Scheme
                </h2>
              </div>
              <div className="p-6 space-y-6">
                <div className="grid grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Primary Color
                    </label>
                    <div className="flex items-center gap-3">
                      <input
                        type="color"
                        value={branding.primary_color}
                        onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })}
                        className="w-12 h-12 rounded-lg border border-gray-300 dark:border-gray-600 cursor-pointer"
                      />
                      <input
                        type="text"
                        value={branding.primary_color}
                        onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })}
                        className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent font-mono"
                      />
                    </div>
                    <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                      Used for buttons, links, and key accents
                    </p>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Secondary Color
                    </label>
                    <div className="flex items-center gap-3">
                      <input
                        type="color"
                        value={branding.secondary_color}
                        onChange={(e) => setBranding({ ...branding, secondary_color: e.target.value })}
                        className="w-12 h-12 rounded-lg border border-gray-300 dark:border-gray-600 cursor-pointer"
                      />
                      <input
                        type="text"
                        value={branding.secondary_color}
                        onChange={(e) => setBranding({ ...branding, secondary_color: e.target.value })}
                        className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent font-mono"
                      />
                    </div>
                    <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                      Used for secondary buttons and highlights
                    </p>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm">
              <div className="p-6 border-b border-gray-200 dark:border-gray-700">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Custom Domain
                </h2>
              </div>
              <div className="p-6 space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Dashboard Domain
                  </label>
                  <input
                    type="text"
                    value={branding.dashboard_domain}
                    onChange={(e) => setBranding({ ...branding, dashboard_domain: e.target.value })}
                    className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                    placeholder="dashboard.yourcompany.com"
                  />
                </div>
                <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                  <div className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <div className="text-sm">
                      <p className="font-medium text-blue-800 dark:text-blue-300">
                        DNS Configuration Required
                      </p>
                      <p className="mt-1 text-blue-700 dark:text-blue-400">
                        Create a CNAME record pointing <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded">dashboard</code> to{" "}
                        <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded">fw-dashboard.example.com</code>
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm">
              <div className="p-6 border-b border-gray-200 dark:border-gray-700">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Support & Legal
                </h2>
              </div>
              <div className="p-6 space-y-6">
                <div className="grid grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Support Email
                    </label>
                    <input
                      type="email"
                      value={branding.support_email}
                      onChange={(e) => setBranding({ ...branding, support_email: e.target.value })}
                      className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                      placeholder="support@yourcompany.com"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Support URL
                    </label>
                    <input
                      type="url"
                      value={branding.support_url}
                      onChange={(e) => setBranding({ ...branding, support_url: e.target.value })}
                      className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                      placeholder="https://help.yourcompany.com"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Custom Email From
                    </label>
                    <input
                      type="text"
                      value={branding.custom_email_from}
                      onChange={(e) => setBranding({ ...branding, custom_email_from: e.target.value })}
                      className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                      placeholder="noreply@yourcompany.com"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Terms of Service URL
                    </label>
                    <input
                      type="url"
                      value={branding.terms_url}
                      onChange={(e) => setBranding({ ...branding, terms_url: e.target.value })}
                      className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                      placeholder="https://yourcompany.com/terms"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Privacy Policy URL
                    </label>
                    <input
                      type="url"
                      value={branding.privacy_url}
                      onChange={(e) => setBranding({ ...branding, privacy_url: e.target.value })}
                      className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                      placeholder="https://yourcompany.com/privacy"
                    />
                  </div>
                </div>

                <div className="flex items-center justify-between py-4 border-t border-gray-200 dark:border-gray-700">
                  <div>
                    <p className="font-medium text-gray-900 dark:text-white">
                      Hide "Powered by FortressWAF"
                    </p>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      Removes the FortressWAF branding from the footer
                    </p>
                  </div>
                  <button
                    onClick={() => setBranding({ ...branding, hide_powered_by: !branding.hide_powered_by })}
                    className={`relative w-12 h-6 rounded-full transition-colors ${
                      branding.hide_powered_by ? "bg-indigo-600" : "bg-gray-300 dark:bg-gray-600"
                    }`}
                  >
                    <span
                      className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${
                        branding.hide_powered_by ? "translate-x-6" : ""
                      }`}
                    />
                  </button>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-between">
              <button
                onClick={handleReset}
                className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              >
                Reset to Default
              </button>
              <div className="flex items-center gap-3">
                {saved && (
                  <span className="text-green-600 dark:text-green-400 text-sm font-medium">
                    Changes saved successfully
                  </span>
                )}
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="px-6 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:bg-indigo-400 text-white font-medium rounded-lg transition-colors flex items-center gap-2"
                >
                  {saving ? (
                    <>
                      <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                      </svg>
                      Saving...
                    </>
                  ) : (
                    <>
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                      Save Branding
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>

          <div className="lg:col-span-1">
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm sticky top-8">
              <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
                <h3 className="font-semibold text-gray-900 dark:text-white">
                  Live Preview
                </h3>
                <div className="flex items-center gap-1 bg-gray-100 dark:bg-gray-900 rounded-lg p-1">
                  <button
                    onClick={() => setPreviewMode("light")}
                    className={`px-3 py-1 text-sm rounded-md transition-colors ${
                      previewMode === "light"
                        ? "bg-white dark:bg-gray-800 text-gray-900 dark:text-white shadow-sm"
                        : "text-gray-500 dark:text-gray-400"
                    }`}
                  >
                    Light
                  </button>
                  <button
                    onClick={() => setPreviewMode("dark")}
                    className={`px-3 py-1 text-sm rounded-md transition-colors ${
                      previewMode === "dark"
                        ? "bg-white dark:bg-gray-800 text-gray-900 dark:text-white shadow-sm"
                        : "text-gray-500 dark:text-gray-400"
                    }`}
                  >
                    Dark
                  </button>
                </div>
              </div>

              <div
                className={`p-4 min-h-[500px] ${
                  previewMode === "dark" ? "bg-gray-900" : "bg-gray-50"
                }`}
              >
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm overflow-hidden">
                  <div
                    className="h-12 px-4 flex items-center gap-3"
                    style={{
                      backgroundColor: `${branding.primary_color}10`,
                      borderBottom: `2px solid ${branding.primary_color}`,
                    }}
                  >
                    {logoPreview ? (
                      <img src={logoPreview} alt="Logo" className="h-8 object-contain" />
                    ) : (
                      <div
                        className="w-8 h-8 rounded flex items-center justify-center text-white text-sm font-bold"
                        style={{ backgroundColor: branding.primary_color }}
                      >
                        {branding.product_name.charAt(0)}
                      </div>
                    )}
                    <span
                      className="font-semibold text-gray-900 dark:text-white"
                      style={{ color: branding.primary_color }}
                    >
                      {branding.product_name}
                    </span>
                  </div>

                  <div className="p-4 space-y-4">
                    <div className="flex items-center gap-2">
                      <div className="w-24 h-6 bg-gray-200 dark:bg-gray-700 rounded" />
                      <div className="w-16 h-6 bg-gray-200 dark:bg-gray-700 rounded" />
                    </div>

                    <div className="grid grid-cols-3 gap-2">
                      {[1, 2, 3].map((i) => (
                        <div key={i} className="h-20 bg-gray-100 dark:bg-gray-900 rounded-lg" />
                      ))}
                    </div>

                    <div className="space-y-2">
                      {[1, 2, 3].map((i) => (
                        <div key={i} className="h-10 bg-gray-100 dark:bg-gray-900 rounded-lg" />
                      ))}
                    </div>

                    <button
                      className="w-full py-2 text-white font-medium rounded-lg"
                      style={{ backgroundColor: branding.primary_color }}
                    >
                      Primary Button
                    </button>

                    <button
                      className="w-full py-2 text-gray-700 dark:text-gray-300 font-medium rounded-lg border-2"
                      style={{ borderColor: branding.secondary_color, color: branding.secondary_color }}
                    >
                      Secondary Button
                    </button>

                    <div
                      className="h-24 rounded-lg flex items-center justify-center"
                      style={{ backgroundColor: `${branding.primary_color}20` }}
                    >
                      <span
                        className="text-2xl font-bold"
                        style={{ color: branding.primary_color }}
                      >
                        {branding.primary_color}
                      </span>
                    </div>
                  </div>
                </div>

                {!branding.hide_powered_by && (
                  <div className="mt-4 text-center">
                    <span className="text-xs text-gray-400 dark:text-gray-500">
                      Powered by{" "}
                      <span className="font-medium">FortressWAF</span>
                    </span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}