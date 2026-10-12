export interface BotAccount { source_protocol: string; id: string; nickname?: string }

export function botAccounts(bots: BotAccount[]): BotAccount[] {
  const accounts = new Map<string, BotAccount>()
  for (const bot of bots) {
    if (!bot.id) continue
    const key = JSON.stringify([bot.source_protocol, bot.id])
    if (!accounts.has(key) || (!accounts.get(key)?.nickname && bot.nickname)) accounts.set(key, bot)
  }
  return [...accounts.values()]
}

export function botLabel(bots: BotAccount[], protocol: string, id: string): string {
  if (!id) return '待选择机器人账号'
  const bot = bots.find(bot => bot.source_protocol === protocol && bot.id === id && bot.nickname)
  return bot?.nickname ? `${bot.nickname}（${id}）` : id
}
