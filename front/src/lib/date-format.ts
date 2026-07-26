const datePattern = /^(\d{4})-(\d{2})-(\d{2})$/;
const dateTimePattern = /^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?/;

const formatLocalDate = (date: Date) => {
  const yyyy = String(date.getFullYear());
  const mm = String(date.getMonth() + 1).padStart(2, "0");
  const dd = String(date.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
};

export const todayLocalDate = () => formatLocalDate(new Date());

export const formatPublishedDate = (value: string) => {
  const trimmed = value.trim();
  if (datePattern.test(trimmed)) {
    return trimmed;
  }

  const match = trimmed.match(dateTimePattern);
  if (!match) {
    return trimmed;
  }

  const [, yyyy, mm, dd, hh, min, ss = "00"] = match;
  const utcDate = new Date(Date.UTC(Number(yyyy), Number(mm) - 1, Number(dd), Number(hh), Number(min), Number(ss)));
  if (Number.isNaN(utcDate.getTime())) {
    return trimmed;
  }
  return formatLocalDate(utcDate);
};

export const normalizePublishedDateForSave = (value: string) => formatPublishedDate(value);
