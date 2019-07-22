Remplacer un partenaire
=======================

.. http:put:: /api/partners/(partner)

   Remplace le partenaire portant le nom `partner` avec les informations renseignées
   en format JSON dans le corps de la requête. Les champs non-spécifiés seront remplacés
   par leur valeur par défaut.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Request JSON Object:

       * **Name** (*string*) - Le nom du partenaire
       * **Address** (*string*) - L'address (IP ou DNS) du partenaire
       * **Port** (*int*) - Le port sur lequel le partenaire écoute
       * **Type** (*[sftp|http]*) - Le type de partenaire

   :Example:
       .. code-block:: http

          PUT /api/partners/partenaire1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 88

          {
            "Name": "partenaire1b",
            "Addresse": "waarp.org",
            "Port": 80,
            "Type": "http"
          }


   **Réponse**

   :statuscode 201: Le partenaire a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du partenaire sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resheader Location: Le chemin d'accès au partenaire modifié

   :Example:
       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/partners/partenaire1b