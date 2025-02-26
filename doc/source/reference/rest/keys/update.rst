Modifier une clé cryptographique
================================

.. http:patch:: /api/keys/(string:key_name)

   Modifie une clé cryptographique existante avec les informations renseignées en
   format JSON dans le corps de la requête. Les paramètres omis resteront
   inchangés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nouveau nom de la clé cryptographique.
   :reqjson string type: Le type de la clé cryptographique. Les valeurs acceptées
      sont :

      - ``AES`` pour les clés de (dé)chiffrement AES
      - ``HMAC`` pour les clés de signature HMAC
      - ``PGP-PUBLIC`` pour les clés PGP publiques
      - ``PGP-PRIVATE`` pour les clés PGP privées
   :reqjson string key: La représentation textuelle de la clé. Si la clé n'est
      pas nativement en format textuel, celle-ci doit être convertie en Base64
      avant son envoi.

   :statuscode 201: La clé a été modifiée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de la requête sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La clé demandée n'existe pas

   :resheader Location: Le nouveau chemin d'accès à la clé mise à jour


   **Exemple de requête**

   .. code-block:: http

      PATCH https://my_waarp_gateway.net/api/keys/aes-key HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 107

      {
        "name": "new-aes-key",
        "type": "AES",
        "privateKey": "ABCDEFHIJKLabcdefhijkl0123456789"
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/keys/new-aes-key
