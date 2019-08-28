Remplacer un partenaire
=======================

.. http:put:: /api/partners/(int:partner_id)

   Remplace le partenaire portant le numéro ``partner_id`` par celui renseigné
   en format JSON dans le corps de la requête. Les champs non-spécifiés seront
   remplacés par leur valeur par défaut.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson number ID: L'identifiant unique du partenaire
   :reqjson string Name: Le nom du partenaire
   :reqjson string Address: L'address (IP ou DNS) du partenaire
   :reqjson number Port: Le port sur lequel le partenaire écoute
   :reqjson [sftp] Type: Le type de partenaire

   **Exemple de requête**

       .. code-block:: http

          PUT /api/partners/1234 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 101

          {
            "ID": 2345,
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
          Location: /api/partners/2345