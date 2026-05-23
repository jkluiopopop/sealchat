export const SMART_LINK_NODE_TYPE = 'smartLink';
export const SMART_LINK_DATA_ATTR = 'data-smart-link';
export const SMART_LINK_IMAGE_ROLE_ATTR = 'data-smart-link-role';
export const SMART_LINK_TEXT_IMAGE_ROLE = 'text-image';

export type SmartLinkTextType = 'text' | 'image';
export type SmartLinkUrlType = 'url' | 'image';

export interface SmartLinkAttrs {
  textType: SmartLinkTextType;
  textValue: string;
  urlType: SmartLinkUrlType;
  urlValue: string;
  target: '_self' | '_blank';
}

const normalizeTextType = (value: unknown): SmartLinkTextType => (
  value === 'image' ? 'image' : 'text'
);

const normalizeUrlType = (value: unknown): SmartLinkUrlType => (
  value === 'image' ? 'image' : 'url'
);

const normalizeTarget = (value: unknown): '_self' | '_blank' => (
  value === '_blank' ? '_blank' : '_self'
);

export function normalizeSmartLinkAttrs(input: Partial<SmartLinkAttrs> | null | undefined): SmartLinkAttrs | null {
  if (!input || typeof input !== 'object') {
    return null;
  }

  const textType = normalizeTextType(input.textType);
  const urlType = normalizeUrlType(input.urlType);
  const textValue = String(input.textValue || '').trim();
  const urlValue = String(input.urlValue || '').trim();
  const target = normalizeTarget(input.target);

  if (!textValue || !urlValue) {
    return null;
  }

  return {
    textType,
    textValue,
    urlType,
    urlValue,
    target,
  };
}

export function isSmartLinkNode(node: any): boolean {
  if (!node) {
    return false;
  }
  if (typeof node.type === 'string') {
    return node.type === SMART_LINK_NODE_TYPE;
  }
  return node.type?.name === SMART_LINK_NODE_TYPE;
}

export function smartLinkToPlainText(input: Partial<SmartLinkAttrs> | null | undefined): string {
  const attrs = normalizeSmartLinkAttrs(input);
  if (!attrs) {
    return '';
  }

  const textSide = attrs.textType === 'image'
    ? '[图片链接文本]'
    : attrs.textValue;
  const urlSide = attrs.urlType === 'image'
    ? '[图片链接目标]'
    : attrs.urlValue;

  return `${textSide} -> ${urlSide}`;
}

export function resolveSmartLinkDisplayText(input: Partial<SmartLinkAttrs> | null | undefined): string {
  const attrs = normalizeSmartLinkAttrs(input);
  if (!attrs) {
    return '';
  }
  if (attrs.textType === 'image') {
    return '[图片链接]';
  }
  return attrs.textValue;
}

export function isSmartLinkImageTarget(input: Partial<SmartLinkAttrs> | null | undefined): boolean {
  return normalizeSmartLinkAttrs(input)?.urlType === 'image';
}

export function isSmartLinkImageText(input: Partial<SmartLinkAttrs> | null | undefined): boolean {
  return normalizeSmartLinkAttrs(input)?.textType === 'image';
}
