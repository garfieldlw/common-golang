package es

import "fmt"

var UserSetting = fmt.Sprintf("{\"settings\":%s,\"mappings\":%s}", ANALYSIS, UserMapping)

const UserMapping = "{\"properties\": {" +
	"    \"id\":{" +
	"        \"type\":\"keyword\"" +
	"    }," +
	"    \"nickname\":{" +
	"        \"type\":\"text\"," +
	"        \"analyzer\":\"ik_max_word_pinyin\"," +
	"        \"search_analyzer\":\"ik_smart_pinyin\"" +
	"    }," +
	"    \"avatar\":{" +
	"        \"type\":\"text\"" +
	"    }," +
	"    \"create_time\":{" +
	"        \"type\":\"long\"" +
	"    }," +
	"    \"status\":{" +
	"        \"type\":\"keyword\"" +
	"    }," +
	"    \"update_time\":{" +
	"        \"type\":\"long\"" +
	"    }" +
	"}}"

const ANALYSIS = "{" +
	"    \"analysis\":{" +
	"        \"analyzer\":{" +
	"            \"ik_smart_pinyin\":{" +
	"                \"type\":\"custom\"," +
	"                \"tokenizer\":\"ik_smart\"," +
	"                \"filter\":[" +
	"                    \"special_characters_filter\"," +
	"                    \"pinyin_filter\"" +
	"                ]" +
	"            }," +
	"            \"ik_max_word_pinyin\":{" +
	"                \"type\":\"custom\"," +
	"                \"tokenizer\":\"ik_max_word\"," +
	"                \"filter\":[" +
	"                    \"special_characters_filter\"," +
	"                    \"pinyin_filter\"" +
	"                ]" +
	"            }" +
	"        }," +
	"        \"filter\":{" +
	"            \"pinyin_filter\":{" +
	"                \"type\":\"pinyin\"," +
	"                \"keep_first_letter\":true," +
	"                \"keep_full_pinyin\":true," +
	"                \"keep_none_chinese\":true," +
	"                \"keep_none_chinese_together\":true," +
	"                \"keep_none_chinese_in_first_letter\":true," +
	"                \"keep_none_chinese_in_joined_full_pinyin\":true," +
	"                \"none_chinese_pinyin_tokenize\":true," +
	"                \"keep_original\":true," +
	"                \"lowercase\":true," +
	"                \"trim_whitespace\":true," +
	"                \"ignore_pinyin_offset\":false," +
	"                \"limit_first_letter_length\":16" +
	"            }," +
	"            \"special_characters_filter\":{" +
	"                \"pattern\":\"\\\\p{Punct}\", " +
	"                \"type\":\"pattern_replace\"," +
	"                \"replacement\":\" \"" +
	"            }" +
	"        }" +
	"    }" +
	"}"
