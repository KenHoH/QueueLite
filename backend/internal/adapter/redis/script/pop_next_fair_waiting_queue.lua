local expected = {'string', 'zset', 'zset', 'zset'}
for i,key in ipairs(KEYS) do
  local kind = redis.call('TYPE', key).ok
  if kind ~= 'none' and kind ~= expected[i] then
    return redis.error_reply('invalid waiting queue key type')
  end
end
local streak = tonumber(redis.call('GET', KEYS[1]) or '0')
if not streak or streak < 0 or streak ~= math.floor(streak) then
  return redis.error_reply('invalid priority streak')
end
local limit = tonumber(ARGV[1])
local function pop(key, priority)
  local item = redis.call('ZPOPMIN', key, 1)
  if #item == 0 then return nil end
  if priority then
    redis.call('SET', KEYS[1], math.min(streak + 1, limit), 'EX', 172800)
  else
    redis.call('SET', KEYS[1], 0, 'EX', 172800)
  end
  return {key, item[1], item[2], tostring(streak)}
end
if streak < limit then
  local item = pop(KEYS[2], true)
  if item then return item end
end
-- Legacy waiting entries retain the upstream normal-queue fallback behavior.
local item = pop(KEYS[3], false) or pop(KEYS[4], false)
if item then return item end
if streak >= limit then
  item = pop(KEYS[2], true)
  if item then return item end
end
return {}
