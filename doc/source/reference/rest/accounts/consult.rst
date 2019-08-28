Consulter un compte
===================

.. http:get:: /api/accounts/(int:account_id)

   Renvoie le compte de l'utilisateur portant le numéro ``account_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          GET /api/accounts/1234 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Le compte a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas

   :resjson number ID: L'identifiant unique du compte
   :resjson number PartnerID: L'identifiant unique du partenaire auquel le compte est rattaché
   :resjson string Username: Le nom d'utilisateur du compte

   :Example:
       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 59

          {
            "ID": 1234,
            "PartnerID": 12345,
            "Name": "partenaire1"
          }