type TiptapCoreModule = typeof import('@tiptap/core');

export type PerformanceEffect = 'shake' | 'wave' | 'rainbow' | 'glitch' | 'blink';
export type PerformanceEnterMode = 'normal' | 'blur' | 'typewriter';
export type PerformanceScale = 'shout' | 'whisper';

export interface PerformanceMarkAttrs {
  effect?: PerformanceEffect | null;
  enterMode?: PerformanceEnterMode | null;
  enterSpeed?: number | null;
  toneIntensity?: number | null;
  scale?: PerformanceScale | null;
}

export const normalizePerformanceEffect = (value: unknown): PerformanceEffect | null => {
  const raw = String(value || '').trim();
  if (raw === 'blur-in') {
    return 'blink';
  }
  if (raw === 'shake' || raw === 'wave' || raw === 'rainbow' || raw === 'glitch' || raw === 'blink') {
    return raw;
  }
  return null;
};

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    performance: {
      setPerformance: (attrs: PerformanceMarkAttrs) => ReturnType;
      unsetPerformance: () => ReturnType;
    };
  }
}

export const createPerformanceExtension = ({
  Mark,
  mergeAttributes,
}: Pick<TiptapCoreModule, 'Mark' | 'mergeAttributes'>) => Mark.create({
  name: 'performance',

  addAttributes() {
    return {
      effect: {
        default: null,
        parseHTML: (element: HTMLElement) => normalizePerformanceEffect(element.getAttribute('data-performance-effect')),
        renderHTML: (attributes: PerformanceMarkAttrs) => {
          const effect = normalizePerformanceEffect(attributes.effect);
          return effect ? { 'data-performance-effect': effect } : {};
        },
      },
      enterMode: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-performance-enter-mode') || null,
        renderHTML: (attributes: PerformanceMarkAttrs) => {
          const mode = String(attributes.enterMode || '').trim();
          return mode ? { 'data-performance-enter-mode': mode } : {};
        },
      },
      enterSpeed: {
        default: null,
        parseHTML: (element: HTMLElement) => {
          const raw = element.getAttribute('data-performance-enter-speed');
          const numeric = raw == null || raw === '' ? NaN : Number(raw);
          return Number.isFinite(numeric) ? numeric : null;
        },
        renderHTML: (attributes: PerformanceMarkAttrs) => {
          const numeric = Number(attributes.enterSpeed);
          return Number.isFinite(numeric) ? { 'data-performance-enter-speed': String(numeric) } : {};
        },
      },
      toneIntensity: {
        default: null,
        parseHTML: (element: HTMLElement) => {
          const raw = element.getAttribute('data-performance-tone-intensity');
          const numeric = raw == null || raw === '' ? NaN : Number(raw);
          return Number.isFinite(numeric) ? numeric : null;
        },
        renderHTML: (attributes: PerformanceMarkAttrs) => {
          const numeric = Number(attributes.toneIntensity);
          return Number.isFinite(numeric) ? { 'data-performance-tone-intensity': String(numeric) } : {};
        },
      },
      scale: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-performance-scale') || null,
        renderHTML: (attributes: PerformanceMarkAttrs) => {
          const scale = String(attributes.scale || '').trim();
          return scale ? { 'data-performance-scale': scale } : {};
        },
      },
    };
  },

  parseHTML() {
    return [
      { tag: 'span[data-performance-effect]' },
      { tag: 'span[data-performance-enter-mode]' },
      { tag: 'span[data-performance-enter-speed]' },
      { tag: 'span[data-performance-tone-intensity]' },
      { tag: 'span[data-performance-scale]' },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    const effect = normalizePerformanceEffect(HTMLAttributes.effect);
    const enterMode = String(HTMLAttributes.enterMode || '').trim();
    const enterSpeed = Number(HTMLAttributes.enterSpeed);
    const toneIntensity = Number(HTMLAttributes.toneIntensity);
    const scale = String(HTMLAttributes.scale || '').trim();
    const classNames = ['tiptap-performance'];
    const styleVars: Record<string, string> = {};
    if (effect) {
      classNames.push(`fx-${effect}`);
    }
    if (enterMode) {
      classNames.push(`enter-${enterMode}`);
    }
    if (Number.isFinite(enterSpeed)) {
      styleVars['--performance-enter-speed'] = String(enterSpeed);
    }
    if (Number.isFinite(toneIntensity)) {
      styleVars['--performance-tone-intensity'] = String(toneIntensity);
    }
    if (scale) {
      classNames.push(`scale-${scale}`);
    }
    return [
      'span',
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes, {
        class: classNames.join(' '),
        style: Object.entries(styleVars).map(([key, value]) => `${key}: ${value}`).join('; '),
      }),
      0,
    ];
  },

  addCommands() {
    return {
      setPerformance:
        (attrs: PerformanceMarkAttrs) =>
        ({ commands }) => {
          const nextAttrs = {
            effect: normalizePerformanceEffect(attrs.effect),
            enterMode: attrs.enterMode || null,
            enterSpeed: Number.isFinite(Number(attrs.enterSpeed)) ? Number(attrs.enterSpeed) : null,
            toneIntensity: Number.isFinite(Number(attrs.toneIntensity)) ? Number(attrs.toneIntensity) : null,
            scale: attrs.scale || null,
          };
          return commands.setMark(this.name, nextAttrs);
        },
      unsetPerformance:
        () =>
        ({ commands }) =>
          commands.unsetMark(this.name),
    };
  },
});
