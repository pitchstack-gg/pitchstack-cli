package cardsdb

import "testing"

func TestParseCardSearchQuery(t *testing.T) {
	params := ParseCardSearchQuery(`class:Wizard type:Action keyword:"go again" pitch>=2 set:ROS rarity:Majestic legal:cc color:red double:true scar`)

	if params.Class != "Wizard" || params.Type != "Action" || params.Keyword != "go again" || params.Pitch != ">=2" {
		t.Fatalf("parsed core filters = %#v", params)
	}
	if params.SetCode != "ROS" || params.Rarity != "Majestic" || params.SearchTerm != "scar" {
		t.Fatalf("parsed printing/free filters = %#v", params)
	}
	if params.ColorIdentity != "COLOR_IDENTITY_RED" || params.IsDoubleFaced == nil || !*params.IsDoubleFaced {
		t.Fatalf("parsed special filters = %#v", params)
	}
	if params.CCLegal == nil || !*params.CCLegal {
		t.Fatalf("CCLegal = %#v, want true", params.CCLegal)
	}
}
