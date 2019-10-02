Consulter un partenaire
=======================

.. http:get:: /api/partners/(int:partner_id)

   Renvoie les informations du partenaire portant l'identifiant ``partner_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/partners/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Les informations du partenaire ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resjson number id: L'identifiant unique du partenaire
   :resjson string name: Le nom du partenaire
   :resjson string protocol: Le protocole utilisé par le partenaire
   :resjson string protoConfig: La configuration du partenaire encodé dans une
      chaîne de caractères au format JSON.

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 97

          {
            "id": 1,
            "name": "waarp_sftp",
            "protocol": "sftp",
            "protoConfig": "{\"address\":\"waarp.org\",\"port\":21}
          }