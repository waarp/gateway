Ajouter une règle
=================

.. http:post:: /api/rules

   Ajoute une nouvelle règle avec les informations renseignées en format JSON dans
   le corps de la requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom de la règle
   :reqjson string comment: Un commentaire optionnel à propos de la règle (description...)
   :reqjson bool isSend: Si vrai, la règle peut être utilisée lors de l'envoi de fichiers,
                         si faux, la règle peut être utilisée lors de la réception de fichiers
   :reqjson string path: Le chemin de destination du fichier

   **Exemple de requête**

       .. code-block:: http

          POST https://my_waarp_gateway.net/api/rules HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 119

          {
            "name": "règle exemple envoi",
            "comment": "ceci est un exemple de règle d'envoi",
            "isSend": true
            "path": "/chemin/distant/de/destination/du/fichier"
          }

   **Réponse**

   :statuscode 201: La règle a été créée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de la règle sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès de la nouvelle règle créée

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/rules/1
