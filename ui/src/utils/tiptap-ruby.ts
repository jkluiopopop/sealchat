type TiptapCoreModule = typeof import('@tiptap/core');

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    ruby: {
      setRuby: (rubyText: string) => ReturnType;
      unsetRuby: () => ReturnType;
    };
  }
}

export interface RubyOptions {
  HTMLAttributes: Record<string, unknown>;
}

const buildRubyStyleVariableString = (attributes: Record<string, any>) => {
  const variables: string[] = [];
  const pushVar = (name: string, value: unknown) => {
    const normalized = String(value || '').trim();
    if (!normalized) {
      return;
    }
    variables.push(`${name}: ${normalized}`);
  };
  pushVar('--ruby-base-font-family', attributes.rubyBaseFontFamily);
  pushVar('--ruby-rt-font-family', attributes.rubyRtFontFamily);
  pushVar('--ruby-base-font-size', attributes.rubyBaseFontSize);
  pushVar('--ruby-font-family', attributes.rubyFontFamily);
  pushVar('--ruby-font-size', attributes.rubyFontSize);
  pushVar('--ruby-rt-font-size', attributes.rubyRtFontSize);
  pushVar('--ruby-color', attributes.rubyColor);
  pushVar('--ruby-font-weight', attributes.rubyFontWeight);
  pushVar('--ruby-font-style', attributes.rubyFontStyle);
  pushVar('--ruby-rt-scale', attributes.rubyRtScale);
  pushVar('--ruby-text-decoration', attributes.rubyTextDecoration);
  pushVar('--ruby-background-color', attributes.rubyBackgroundColor);
  const existingStyle = String(attributes.style || '').trim();
  return [existingStyle, variables.join('; ')].filter(Boolean).join('; ');
};

export const createRubyExtension = ({
  Mark,
  mergeAttributes,
}: Pick<TiptapCoreModule, 'Mark' | 'mergeAttributes'>) => Mark.create<RubyOptions>({
  name: 'ruby',

  addOptions() {
    return {
      HTMLAttributes: {},
    };
  },

  addAttributes() {
    return {
      rubyText: {
        default: null,
        parseHTML: (element: HTMLElement) => {
          const rubyText = element.querySelector('rt')?.textContent || '';
          return rubyText.trim() || null;
        },
        renderHTML: (attributes: Record<string, any>) => {
          const rubyText = String(attributes.rubyText || '').trim();
          if (!rubyText) {
            return {};
          }
          return {
            'data-ruby-text': rubyText,
          };
        },
      },
      rubyFontFamily: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-font-family') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyFontFamily || '').trim();
          return value ? { 'data-ruby-font-family': value } : {};
        },
      },
      rubyBaseFontFamily: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-base-font-family') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyBaseFontFamily || '').trim();
          return value ? { 'data-ruby-base-font-family': value } : {};
        },
      },
      rubyRtFontFamily: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-rt-font-family') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyRtFontFamily || '').trim();
          return value ? { 'data-ruby-rt-font-family': value } : {};
        },
      },
      rubyFontSize: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-font-size') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyFontSize || '').trim();
          return value ? { 'data-ruby-font-size': value } : {};
        },
      },
      rubyBaseFontSize: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-base-font-size') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyBaseFontSize || '').trim();
          return value ? { 'data-ruby-base-font-size': value } : {};
        },
      },
      rubyRtFontSize: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-rt-font-size') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyRtFontSize || '').trim();
          return value ? { 'data-ruby-rt-font-size': value } : {};
        },
      },
      rubyColor: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-color') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyColor || '').trim();
          return value ? { 'data-ruby-color': value } : {};
        },
      },
      rubyFontWeight: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-font-weight') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyFontWeight || '').trim();
          return value ? { 'data-ruby-font-weight': value } : {};
        },
      },
      rubyFontStyle: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-font-style') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyFontStyle || '').trim();
          return value ? { 'data-ruby-font-style': value } : {};
        },
      },
      rubyRtScale: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-rt-scale') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyRtScale || '').trim();
          return value ? { 'data-ruby-rt-scale': value } : {};
        },
      },
      rubyFontAssetId: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-platform-font-id') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyFontAssetId || '').trim();
          return value ? { 'data-platform-font-id': value } : {};
        },
      },
      rubyBaseFontAssetId: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-base-font-asset-id') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyBaseFontAssetId || '').trim();
          return value ? { 'data-ruby-base-font-asset-id': value } : {};
        },
      },
      rubyRtFontAssetId: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-rt-font-asset-id') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyRtFontAssetId || '').trim();
          return value ? { 'data-ruby-rt-font-asset-id': value } : {};
        },
      },
      rubyPlatformFontFamily: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-platform-font-family') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyPlatformFontFamily || '').trim();
          return value ? { 'data-platform-font-family': value } : {};
        },
      },
      rubyBasePlatformFontFamily: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-base-platform-font-family') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyBasePlatformFontFamily || '').trim();
          return value ? { 'data-ruby-base-platform-font-family': value } : {};
        },
      },
      rubyRtPlatformFontFamily: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-rt-platform-font-family') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyRtPlatformFontFamily || '').trim();
          return value ? { 'data-ruby-rt-platform-font-family': value } : {};
        },
      },
      rubyTextDecoration: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-text-decoration') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyTextDecoration || '').trim();
          return value ? { 'data-ruby-text-decoration': value } : {};
        },
      },
      rubyBackgroundColor: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-ruby-background-color') || null,
        renderHTML: (attributes: Record<string, any>) => {
          const value = String(attributes.rubyBackgroundColor || '').trim();
          return value ? { 'data-ruby-background-color': value } : {};
        },
      },
      rubySpoiler: {
        default: null,
        parseHTML: (element: HTMLElement) => {
          const value = element.getAttribute('data-ruby-spoiler');
          return value === 'true' ? 'true' : null;
        },
        renderHTML: (attributes: Record<string, any>) => {
          return attributes.rubySpoiler === 'true' ? { 'data-ruby-spoiler': 'true' } : {};
        },
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: 'ruby',
      },
      {
        tag: 'span[data-ruby-text]',
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      'span',
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes, {
        class: 'tiptap-ruby',
        style: buildRubyStyleVariableString(HTMLAttributes),
      }),
      0,
    ];
  },

  addCommands() {
    return {
      setRuby:
        (rubyText: string) =>
        ({ commands }) => {
          const normalized = String(rubyText || '').trim();
          if (!normalized) {
            return commands.unsetMark(this.name);
          }
          return commands.setMark(this.name, { rubyText: normalized });
        },
      unsetRuby:
        () =>
        ({ commands }) =>
          commands.unsetMark(this.name),
    };
  },
});
