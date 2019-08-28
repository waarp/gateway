Consulter un partenaire
=======================

.. http:get:: /api/partners/(int:partner_id)

   Renvoie portant l'identifiant ``partner_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET /api/partners/1234 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Le partenaire a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resjson number ID: L'identifiant unique du partenaire
   :resjson string Name: Le nom du partenaire
   :resjson string Address: L'address (IP ou DNS) du partenaire
   :resjson number Port: Le port sur lequel le partenaire écoute
   :resjson [sftp] Type: Le type de partenaire

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 101

          {
            "ID": 1234,
            "Name": "partenaire1",
            "Addresse": "waarp.fr",
            "Port": 21,
            "Type": "sftp"
          }