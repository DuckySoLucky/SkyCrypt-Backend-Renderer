package minecraftblockrenderer

import (
	"fmt"
	nbt "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/NBT"
	"github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/data"
	"image"
	"strings"
)

func (renderer *MinecraftBlockRenderer) RenderItemObject(item any, options *BlockRenderOptions) (*image.RGBA, error) {
	normalized, err := data.NormalizeItemInput(item)
	if err != nil {
		return nil, err
	}

	itemName, itemData := renderer.prepareItemObjectRender(normalized)
	if strings.TrimSpace(itemName) == "" {
		return nil, fmt.Errorf("unable to resolve item id from input type %T", item)
	}

	effectiveOptions := mergeItemObjectOptions(options, itemData)
	itemName = renderer.resolvePackedSkyblockItemObjectName(normalized, itemName, effectiveOptions)
	effectiveOptions = renderer.normalizePackedSkyblockItemObjectOptions(normalized, effectiveOptions)
	return renderer.RenderItem(itemName, effectiveOptions.ItemData, effectiveOptions), nil
}

func (renderer *MinecraftBlockRenderer) RenderItemObjectWithResourceId(item any, options *BlockRenderOptions) (*RenderedResource, error) {
	normalized, err := data.NormalizeItemInput(item)
	if err != nil {
		return nil, err
	}

	itemName, itemData := renderer.prepareItemObjectRender(normalized)
	if strings.TrimSpace(itemName) == "" {
		return nil, fmt.Errorf("unable to resolve item id from input type %T", item)
	}

	effectiveOptions := mergeItemObjectOptions(options, itemData)
	itemName = renderer.resolvePackedSkyblockItemObjectName(normalized, itemName, effectiveOptions)
	effectiveOptions = renderer.normalizePackedSkyblockItemObjectOptions(normalized, effectiveOptions)
	return renderer.RenderGuiItemWithResourceId(itemName, effectiveOptions), nil
}

func (renderer *MinecraftBlockRenderer) RenderAnimatedItemObjectWithResourceId(item any, options *BlockRenderOptions) (*AnimatedRenderedResource, error) {
	normalized, err := data.NormalizeItemInput(item)
	if err != nil {
		return nil, err
	}

	itemName, itemData := renderer.prepareItemObjectRender(normalized)
	if strings.TrimSpace(itemName) == "" {
		return nil, fmt.Errorf("unable to resolve item id from input type %T", item)
	}

	effectiveOptions := mergeItemObjectOptions(options, itemData)
	itemName = renderer.resolvePackedSkyblockItemObjectName(normalized, itemName, effectiveOptions)
	effectiveOptions = renderer.normalizePackedSkyblockItemObjectOptions(normalized, effectiveOptions)
	return renderer.RenderAnimatedGuiItemWithResourceId(itemName, effectiveOptions)
}

func (renderer *MinecraftBlockRenderer) RenderSkyBlockItemID(skyBlockItemID string, options *BlockRenderOptions) (*RenderedResource, error) {
	target, effectiveOptions, err := renderer.prepareSkyBlockItemIDRender(skyBlockItemID, options)
	if err != nil {
		return nil, err
	}
	return renderer.RenderGuiItemWithResourceId(target, effectiveOptions), nil
}

func (renderer *MinecraftBlockRenderer) RenderItemNBT(item any, options *BlockRenderOptions) (*RenderedResource, error) {
	return renderer.RenderItemObjectWithResourceId(normalizeNbtRenderInput(item), options)
}

func (renderer *MinecraftBlockRenderer) RenderAnimatedSkyBlockItemID(skyBlockItemID string, options *BlockRenderOptions) (*AnimatedRenderedResource, error) {
	target, effectiveOptions, err := renderer.prepareSkyBlockItemIDRender(skyBlockItemID, options)
	if err != nil {
		return nil, err
	}
	return renderer.RenderAnimatedGuiItemWithResourceId(target, effectiveOptions)
}

func (renderer *MinecraftBlockRenderer) RenderAnimatedItemNBT(item any, options *BlockRenderOptions) (*AnimatedRenderedResource, error) {
	return renderer.RenderAnimatedItemObjectWithResourceId(normalizeNbtRenderInput(item), options)
}

func (renderer *MinecraftBlockRenderer) ComputeResourceIdFromSkyBlockItemID(skyBlockItemID string, options *BlockRenderOptions) (*ResourceIdResult, error) {
	target, effectiveOptions, err := renderer.prepareSkyBlockItemIDRender(skyBlockItemID, options)
	if err != nil {
		return nil, err
	}
	rendererForOptions, forwardedOptions := renderer.ResolveRendererForOptions(*effectiveOptions)
	return rendererForOptions.ComputeResourceIdInternal(target, forwardedOptions, nil), nil
}

func (renderer *MinecraftBlockRenderer) ComputeResourceIdFromNBT(item any, options *BlockRenderOptions) (*ResourceIdResult, error) {
	return renderer.ComputeResourceIdFromItemObject(normalizeNbtRenderInput(item), options)
}

func (renderer *MinecraftBlockRenderer) ComputeResourceIdFromItemObject(item any, options *BlockRenderOptions) (*ResourceIdResult, error) {
	normalized, err := data.NormalizeItemInput(item)
	if err != nil {
		return nil, err
	}

	itemName, itemData := renderer.prepareItemObjectRender(normalized)
	if strings.TrimSpace(itemName) == "" {
		return nil, fmt.Errorf("unable to resolve item id from input type %T", item)
	}

	effectiveOptions := mergeItemObjectOptions(options, itemData)
	itemName = renderer.resolvePackedSkyblockItemObjectName(normalized, itemName, effectiveOptions)
	effectiveOptions = renderer.normalizePackedSkyblockItemObjectOptions(normalized, effectiveOptions)
	rendererForOptions, forwardedOptions := renderer.ResolveRendererForOptions(*effectiveOptions)
	return rendererForOptions.ComputeResourceIdInternal(itemName, forwardedOptions, nil), nil
}

func (renderer *MinecraftBlockRenderer) prepareSkyBlockItemIDRender(skyBlockItemID string, options *BlockRenderOptions) (string, *BlockRenderOptions, error) {
	if strings.TrimSpace(skyBlockItemID) == "" {
		return "", nil, fmt.Errorf("skyBlockItemID cannot be empty")
	}

	itemData := &data.ItemRenderData{
		CustomData: nbt.NewNbtCompound(map[string]nbt.NbtTag{
			"id": nbt.NewNbtString(skyBlockItemID),
		}),
	}

	effectiveOptions := mergeItemObjectOptions(options, itemData)
	if strings.TrimSpace(effectiveOptions.CustomTextureFallbackItem) == "" {
		effectiveOptions.CustomTextureFallbackItem = "minecraft:player_head"
	}
	target := "minecraft:player_head"
	if len(effectiveOptions.PackIds) > 0 {
		target = "firmskyblock:item/" + renderer.EncodeFirmamentId(skyBlockItemID)
	}
	return target, effectiveOptions, nil
}

func (renderer *MinecraftBlockRenderer) prepareItemObjectRender(normalized *data.NormalizedItemInput) (string, *data.ItemRenderData) {
	itemName := renderer.resolveItemObjectName(normalized)
	itemData := buildItemObjectRenderData(normalized)
	if normalized != nil && normalized.DisplayColor != nil && renderer.ShouldApplyNBTDisplayColor(itemName) {
		itemData = ensureItemRenderData(itemData)
		itemData.Layer0Tint = normalized.DisplayColor
	}
	return itemName, itemData
}

func (renderer *MinecraftBlockRenderer) resolvePackedSkyblockItemObjectName(normalized *data.NormalizedItemInput, itemName string, options *BlockRenderOptions) string {
	if normalized == nil || options == nil || len(options.PackIds) == 0 {
		return itemName
	}
	if strings.TrimSpace(normalized.SkyblockID) == "" {
		return itemName
	}
	if strings.TrimSpace(options.CustomTextureFallbackItem) == "" {
		options.CustomTextureFallbackItem = renderer.resolvePackedSkyblockFallbackItem(normalized, itemName)
	}
	return "firmskyblock:item/" + renderer.EncodeFirmamentId(normalized.SkyblockID)
}

func (renderer *MinecraftBlockRenderer) resolvePackedSkyblockFallbackItem(normalized *data.NormalizedItemInput, itemName string) string {
	if normalized != nil {
		if strings.TrimSpace(normalized.ItemID) != "" {
			return normalized.ItemID
		}
		if normalized.NumericID != nil {
			if mapped, ok := legacyNumericItemID(*normalized.NumericID, normalized.Damage); ok {
				return "minecraft:" + mapped
			}
		}
		if strings.TrimSpace(normalized.ItemModel) != "" {
			return normalized.ItemModel
		}
	}
	skyblockID := ""
	if normalized != nil {
		skyblockID = strings.TrimSpace(normalized.SkyblockID)
	}
	if strings.TrimSpace(itemName) != "" && !strings.EqualFold(itemName, skyblockID) {
		return itemName
	}
	return "minecraft:player_head"
}

func (renderer *MinecraftBlockRenderer) normalizePackedSkyblockItemObjectOptions(normalized *data.NormalizedItemInput, options *BlockRenderOptions) *BlockRenderOptions {
	if normalized == nil || options == nil || len(options.PackIds) == 0 {
		return options
	}
	if strings.TrimSpace(normalized.SkyblockID) == "" || options.ItemData == nil {
		return options
	}

	effectiveOptions := *options
	itemData := *options.ItemData
	if effectiveOptions.CustomTextureFallbackData == nil {
		fallbackData := itemData
		effectiveOptions.CustomTextureFallbackData = &fallbackData
	}
	itemData.Profile = nil
	itemData.Layer0Tint = nil
	itemData.AdditionalLayerTints = nil
	effectiveOptions.ItemData = &itemData
	return &effectiveOptions
}

func (renderer *MinecraftBlockRenderer) resolveItemObjectName(normalized *data.NormalizedItemInput) string {
	if normalized == nil {
		return ""
	}

	if strings.TrimSpace(normalized.ItemModel) != "" {
		model := strings.TrimSpace(normalized.ItemModel)
		if strings.HasPrefix(strings.ToLower(model), "minecraft:") {
			return strings.TrimPrefix(model, "minecraft:")
		}
		return model
	}

	if normalized.NumericID != nil {
		if mapped, ok := legacyNumericItemID(*normalized.NumericID, normalized.Damage); ok {
			return mapped
		}
	}

	if strings.TrimSpace(normalized.ItemID) != "" {
		return normalized.ItemID
	}

	if strings.TrimSpace(normalized.SkyblockID) != "" {
		return normalized.SkyblockID
	}

	return ""
}

func buildItemObjectRenderData(normalized *data.NormalizedItemInput) *data.ItemRenderData {
	if normalized == nil {
		return nil
	}

	itemData := &data.ItemRenderData{}
	if normalized.ExtraAttributes != nil {
		itemData.CustomData = data.DecodedMapToNbtCompound(normalized.ExtraAttributes)
	}
	if itemData.CustomData == nil && normalized.CustomData != nil {
		itemData.CustomData = data.DecodedMapToNbtCompound(normalized.CustomData)
	}
	if normalized.SkullProfile != nil {
		itemData.Profile = skullProfileMapToNbtCompound(normalized.SkullProfile)
	}

	if itemData.CustomData == nil && itemData.Profile == nil && itemData.Layer0Tint == nil {
		return nil
	}

	return itemData
}

func ensureItemRenderData(itemData *data.ItemRenderData) *data.ItemRenderData {
	if itemData != nil {
		return itemData
	}
	return &data.ItemRenderData{}
}

func (renderer *MinecraftBlockRenderer) ShouldApplyNBTDisplayColor(itemName string) bool {
	normalized := renderer.NormalizeItemTextureKey(itemName)
	if strings.TrimSpace(normalized) == "" {
		return false
	}

	if _, found := LegacyDefaultTintOverrides[normalized]; found {
		return true
	}
	return strings.HasPrefix(strings.ToLower(normalized), "leather_")
}

func mergeItemObjectOptions(options *BlockRenderOptions, itemData *data.ItemRenderData) *BlockRenderOptions {
	effective := MergeBlockRenderOptions(options)
	if itemData != nil {
		effective.ItemData = itemData
	}
	return &effective
}

func normalizeNbtRenderInput(item any) any {
	switch typed := item.(type) {
	case *nbt.NbtCompound:
		return nbtCompoundToMap(typed)
	default:
		return item
	}
}

func nbtCompoundToMap(compound *nbt.NbtCompound) map[string]any {
	if compound == nil {
		return nil
	}
	result := make(map[string]any, compound.Count())
	for key, tag := range compound.Items() {
		result[key] = nbtTagToAny(tag)
	}
	return result
}

func nbtTagToAny(tag nbt.NbtTag) any {
	switch typed := tag.(type) {
	case nbt.NbtString:
		return typed.Value
	case *nbt.NbtString:
		return typed.Value
	case nbt.NbtByte:
		return int(typed.Value)
	case *nbt.NbtByte:
		return int(typed.Value)
	case nbt.NbtShort:
		return int(typed.Value)
	case *nbt.NbtShort:
		return int(typed.Value)
	case nbt.NbtInt:
		return int(typed.Value)
	case *nbt.NbtInt:
		return int(typed.Value)
	case nbt.NbtLong:
		return typed.Value
	case *nbt.NbtLong:
		return typed.Value
	case nbt.NbtFloat:
		return typed.Value
	case *nbt.NbtFloat:
		return typed.Value
	case nbt.NbtDouble:
		return typed.Value
	case *nbt.NbtDouble:
		return typed.Value
	case nbt.NbtByteArray:
		return typed.Values
	case *nbt.NbtByteArray:
		return typed.Values
	case nbt.NbtIntArray:
		return typed.Values
	case *nbt.NbtIntArray:
		return typed.Values
	case nbt.NbtLongArray:
		return typed.Values
	case *nbt.NbtLongArray:
		return typed.Values
	case *nbt.NbtCompound:
		return nbtCompoundToMap(typed)
	case *nbt.NbtList:
		items := make([]any, 0, typed.Count())
		for i := 0; i < typed.Count(); i++ {
			items = append(items, nbtTagToAny(typed.At(i)))
		}
		return items
	default:
		return fmt.Sprint(tag)
	}
}

func skullProfileMapToNbtCompound(profile map[string]any) *nbt.NbtCompound {
	value, hasValue := profileString(profile, "value", "Value")
	if !hasValue || strings.TrimSpace(value) == "" {
		return nil
	}

	propertyEntries := map[string]nbt.NbtTag{
		"name":  nbt.NewNbtString("textures"),
		"value": nbt.NewNbtString(value),
	}
	if signature, ok := profileString(profile, "signature", "Signature"); ok && strings.TrimSpace(signature) != "" {
		propertyEntries["signature"] = nbt.NewNbtString(signature)
	}

	propertyCompound := nbt.NewNbtCompound(propertyEntries)
	profileEntries := map[string]nbt.NbtTag{
		"properties": nbt.NewNbtList(nbt.NbtTagTypeCompound, []nbt.NbtTag{propertyCompound}),
	}

	if id, ok := profileString(profile, "id", "Id", "ID"); ok && strings.TrimSpace(id) != "" {
		profileEntries["id"] = nbt.NewNbtString(id)
	}

	return nbt.NewNbtCompound(profileEntries)
}

func profileString(values map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		for actualKey, value := range values {
			if strings.EqualFold(actualKey, key) {
				text := fmt.Sprint(value)
				return text, strings.TrimSpace(text) != ""
			}
		}
	}
	return "", false
}

func legacyNumericItemID(id int, damage int) (string, bool) {
	if id == 35 {
		if wool, ok := legacyWoolByDamage[damage]; ok {
			return wool, true
		}
		return "white_wool", true
	}

	if id == 397 {
		if damage == 3 {
			return "player_head", true
		}
		return "skeleton_skull", true
	}

	if mapped, ok := legacyNumericItems[id]; ok {
		return mapped, true
	}

	return "", false
}

var legacyWoolByDamage = map[int]string{
	0:  "white_wool",
	1:  "orange_wool",
	2:  "magenta_wool",
	3:  "light_blue_wool",
	4:  "yellow_wool",
	5:  "lime_wool",
	6:  "pink_wool",
	7:  "gray_wool",
	8:  "light_gray_wool",
	9:  "cyan_wool",
	10: "purple_wool",
	11: "blue_wool",
	12: "brown_wool",
	13: "green_wool",
	14: "red_wool",
	15: "black_wool",
}

var legacyNumericItems = map[int]string{
	1:   "stone",
	2:   "grass_block",
	3:   "dirt",
	4:   "cobblestone",
	5:   "oak_planks",
	12:  "sand",
	13:  "gravel",
	17:  "oak_log",
	20:  "glass",
	24:  "sandstone",
	41:  "gold_block",
	42:  "iron_block",
	45:  "bricks",
	49:  "obsidian",
	57:  "diamond_block",
	87:  "netherrack",
	89:  "glowstone",
	98:  "stone_bricks",
	103: "melon",
	155: "quartz_block",
	159: "white_terracotta",
	160: "white_stained_glass_pane",
	161: "acacia_leaves",
	162: "acacia_log",
	170: "hay_block",
	171: "white_carpet",
	172: "terracotta",
	173: "coal_block",
	174: "packed_ice",
	175: "sunflower",
	256: "iron_shovel",
	257: "iron_pickaxe",
	258: "iron_axe",
	260: "apple",
	261: "bow",
	262: "arrow",
	263: "coal",
	264: "diamond",
	265: "iron_ingot",
	266: "gold_ingot",
	267: "iron_sword",
	268: "wooden_sword",
	269: "wooden_shovel",
	270: "wooden_pickaxe",
	271: "wooden_axe",
	272: "stone_sword",
	273: "stone_shovel",
	274: "stone_pickaxe",
	275: "stone_axe",
	276: "diamond_sword",
	277: "diamond_shovel",
	278: "diamond_pickaxe",
	279: "diamond_axe",
	280: "stick",
	281: "bowl",
	282: "mushroom_stew",
	283: "golden_sword",
	284: "golden_shovel",
	285: "golden_pickaxe",
	286: "golden_axe",
	287: "string",
	288: "feather",
	289: "gunpowder",
	290: "wooden_hoe",
	291: "stone_hoe",
	292: "iron_hoe",
	293: "diamond_hoe",
	294: "golden_hoe",
	295: "wheat_seeds",
	296: "wheat",
	297: "bread",
	298: "leather_helmet",
	299: "leather_chestplate",
	300: "leather_leggings",
	301: "leather_boots",
	302: "chainmail_helmet",
	303: "chainmail_chestplate",
	304: "chainmail_leggings",
	305: "chainmail_boots",
	306: "iron_helmet",
	307: "iron_chestplate",
	308: "iron_leggings",
	309: "iron_boots",
	310: "diamond_helmet",
	311: "diamond_chestplate",
	312: "diamond_leggings",
	313: "diamond_boots",
	314: "golden_helmet",
	315: "golden_chestplate",
	316: "golden_leggings",
	317: "golden_boots",
	318: "flint",
	319: "porkchop",
	320: "cooked_porkchop",
	322: "golden_apple",
	329: "saddle",
	331: "redstone",
	332: "snowball",
	334: "leather",
	335: "milk_bucket",
	336: "brick",
	337: "clay_ball",
	338: "sugar_cane",
	339: "paper",
	340: "book",
	341: "slime_ball",
	344: "egg",
	345: "compass",
	346: "fishing_rod",
	347: "clock",
	348: "glowstone_dust",
	349: "cod",
	350: "cooked_cod",
	351: "white_dye",
	352: "bone",
	353: "sugar",
	354: "cake",
	357: "cookie",
	359: "shears",
	360: "melon_slice",
	361: "pumpkin_seeds",
	362: "melon_seeds",
	363: "beef",
	364: "cooked_beef",
	365: "chicken",
	366: "cooked_chicken",
	367: "rotten_flesh",
	368: "ender_pearl",
	369: "blaze_rod",
	370: "ghast_tear",
	371: "gold_nugget",
	372: "nether_wart",
	373: "potion",
	374: "glass_bottle",
	375: "spider_eye",
	376: "fermented_spider_eye",
	377: "blaze_powder",
	378: "magma_cream",
	381: "ender_eye",
	382: "glistering_melon_slice",
	383: "pig_spawn_egg",
	384: "experience_bottle",
	385: "fire_charge",
	388: "emerald",
	391: "carrot",
	392: "potato",
	393: "baked_potato",
	394: "poisonous_potato",
	395: "map",
	396: "golden_carrot",
	399: "nether_star",
	400: "pumpkin_pie",
	403: "enchanted_book",
	406: "quartz",
	417: "iron_horse_armor",
	418: "golden_horse_armor",
	419: "diamond_horse_armor",
	420: "lead",
	421: "name_tag",
	423: "mutton",
	424: "cooked_mutton",
	431: "dark_oak_door",
}
