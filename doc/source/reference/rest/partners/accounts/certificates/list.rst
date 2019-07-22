Lister les certificats
======================

.. http:get:: /api/partners/(partner)/accounts/(account)/certificates

   Renvoie une liste des certificats du compte `account` rattaché au partenaire
   nommé `partner` qui remplissent les critères données en paramètres de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les certificats seront triés *(défaut: name)*
   :type sortby: [name]
   :param order: L'ordre dans lequel les certificats sont triés *(défaut: asc)*
   :type order: [asc|desc]

   :Example:
       .. code-block:: http

          GET /api/partners/partenaire1/accounts/utilisateur1/certificates?limit=5 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le certificat, compte ou partenaire demandé n'existe pas

   :Response JSON Object:
       * **Certificates** (*array* of *object*) - La liste des certificats demandés

           * **Name** (*string*) - Le nom du certificat
           * **PrivateKey** (*string*) - La clé privée du certificat
           * **PublicKey** (*string*) - La clé publique du certificat
           * **PrivateCert** (*string*) - Le certificat privé du compte
           * **PublicCert** (*string*) - Le certificat public du compte

   :Example:
       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 377

          {
            "Certificates": [{
              "Name": "certificat1",
              "PrivateKey": "*clé privée*",
              "PublicKey": "*clé publique*",
              "PrivateCert": "*certificat privée*",
              "PublicCert": "*certificat public*"
            },{
              "Name": "certificat2",
              "PrivateKey": "*clé privée*",
              "PublicKey": "*clé publique*",
              "PrivateCert": "*certificat privée*",
              "PublicCert": "*certificat public*"
            }]
          }