// $("qry-btn").click(function () {
//     console.log("Pressed")
// });

// function qry() {
//   console.log("Search button pressed")
//   text = document.getElementById("qry-frm").submit();
//   $.ajax({
//         url   : '/qry' + text
//         method: 'GET'
//   }).done();
// }


query_button = $("#qry-btn")

query_button.click(function () {
  console.log("Search button pressed")
  //   var loadId = $('#load-id').val();

  //   $.ajax({
  //     url   : '/api/document/load/' + loadId,
  //     method: 'GET'
  //   }).done(function (data) {
  //     documentId = parseInt(loadId);
  //     updateIframe(data);
  //     updateInterface();
  //   })
  // });
});