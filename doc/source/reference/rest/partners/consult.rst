Consulter un partenaire
=======================

.. http:get:: /api/partners/(partner)

   Renvoie le partenaire nommé `partner`.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          GET /api/partners/partenaire1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Le partenaire a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :Response JSON Object:

       * **Name** (*string*) - Le nom du partenaire
       * **Address** (*string*) - L'address (IP ou DNS) du partenaire
       * **Port** (*int*) - Le port sur lequel le partenaire écoute
       * **Type** (*[sftp|http]*) - Le type de partenaire

   :Example:
       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 88

          {
            "Name": "partenaire1",
            "Addresse": "waarp.fr",
            "Port": 21,
            "Type": "sftp"
          }